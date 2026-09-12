package teacherservice

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	workspace "github.com/xmz14/lll/backend-go/internal/modules/projects"
	sourcestore "github.com/xmz14/lll/backend-go/internal/modules/sources"
	conversationstore "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/conversation"
	teachergateway "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/gateway"
	websearch "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/websearch"
)

type fakeGateway struct {
	proposalID        string
	objective         string
	sourceRefs        []string
	taskType          string
	practiceRequested bool
}

func (f fakeGateway) Stream(_ context.Context, _ teachergateway.Request, emit func(teachergateway.Event)) error {
	objective := f.objective
	if objective == "" {
		objective = "核查当前结论"
	}
	emit(teachergateway.Event{Type: "text-delta", Delta: "助教已开始。"})
	taskType := f.taskType
	if taskType == "" {
		taskType = "verify"
	}
	emit(teachergateway.Event{Type: "tool-call-ready", ToolCall: &teachergateway.ToolCall{CallID: "call_test", ToolName: "delegate_learning_work", Arguments: map[string]any{"taskType": taskType, "objective": objective, "sourceRefs": f.sourceRefs, "practiceRequested": f.practiceRequested, "proposalMessageId": f.proposalID}}})
	emit(teachergateway.Event{Type: "response-completed"})
	return nil
}

type sourceGateway struct {
	proposalID string
	sourceID   string
}

type fakeTaskAuthorizer struct {
	inputs []DelegationInput
}

func (f *fakeTaskAuthorizer) CreateDelegation(_ string, input DelegationInput) (DelegatedTask, bool, error) {
	f.inputs = append(f.inputs, input)
	return DelegatedTask{ID: "task_test", Status: "queued"}, true, nil
}

func newTestService(gateway teachergateway.Gateway) *Service {
	service := New(gateway)
	service.Authorizer = &fakeTaskAuthorizer{}
	return service
}

func (f sourceGateway) Stream(_ context.Context, _ teachergateway.Request, emit func(teachergateway.Event)) error {
	emit(teachergateway.Event{Type: "tool-call-ready", ToolCall: &teachergateway.ToolCall{CallID: "call_source", ToolName: "delegate_learning_work", Arguments: map[string]any{"taskType": "verify", "objective": "读取讲义并核查当前结论", "sourceRefs": []string{f.sourceID}, "proposalMessageId": f.proposalID}}})
	return nil
}

func TestApprovedDelegationCreatesTask(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	conversation, _ := conversationstore.New("topic")
	proposal, _, err := conversation.AppendMessage("teacher", "completed", "", []conversationstore.Block{{Type: "markdown", Source: "我准备让助教核查结论，会给出证据。是否同意？"}})
	if err != nil {
		t.Fatal(err)
	}
	service := newTestService(fakeGateway{proposalID: proposal.ID})
	var types []string
	if err := service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_test", Content: "同意"}, func(frame StreamFrame) { types = append(types, frame.Type) }); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, kind := range types {
		if kind == "task-accepted" {
			found = true
		}
	}
	if !found {
		t.Fatalf("task not accepted: %v", types)
	}
}

func TestApprovedConsolidationCarriesExplicitPracticeRequest(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	conversation, _ := conversationstore.New("topic")
	_, _, _ = conversation.AppendMessage("teacher", "completed", "", []conversationstore.Block{{Type: "markdown", Source: "我准备让助教沉淀本轮教学稿并同时出题，使用本轮对话；预期产出是引言、正文和练习；学习意义是形成可复习材料。是否同意？"}})
	service := newTestService(fakeGateway{taskType: "consolidate", objective: "沉淀本轮教学稿并同时出题", practiceRequested: true})
	if err := service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_consolidate", Content: "同意"}, func(StreamFrame) {}); err != nil {
		t.Fatal(err)
	}
	inputs := service.Authorizer.(*fakeTaskAuthorizer).inputs
	if len(inputs) != 1 || !inputs[0].PracticeRequested {
		t.Fatalf("practice request was lost: %#v", inputs)
	}
}

func TestApprovedDelegationAcceptsOKAndIgnoresLegacyLeakedMessageID(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	conversation, _ := conversationstore.New("topic")
	_, _, err := conversation.AppendMessage("teacher", "completed", "", []conversationstore.Block{{Type: "markdown", Source: "[messageId=msg_stale]\n\n我准备让助教整理 MapReduce 论文精读材料，包含调度、容错和权衡分析。是否同意？"}})
	if err != nil {
		t.Fatal(err)
	}
	service := newTestService(fakeGateway{proposalID: "msg_stale", objective: "整理 MapReduce 论文精读材料，包含调度、容错和权衡分析", sourceRefs: []string{"public:MapReduce paper"}})
	accepted := false
	if err := service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_legacy_approval", Content: "ok，你不需要规定太多提示词和约束。"}, func(frame StreamFrame) {
		accepted = accepted || frame.Type == "task-accepted"
	}); err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("short OK approval with a legacy leaked message ID did not create a task")
	}
}

func TestLegacyMessageIDPrefixIsNotSentBackToProvider(t *testing.T) {
	source := "[messageId=msg_stale]\n\n真正的教师内容"
	if got := stripLegacyMessageIDPrefix(source); got != "真正的教师内容" {
		t.Fatalf("legacy prefix was not stripped: %q", got)
	}
	ordinary := "正文里讨论 messageId 的概念"
	if got := stripLegacyMessageIDPrefix(ordinary); got != ordinary {
		t.Fatalf("ordinary content changed: %q", got)
	}
}

func TestDelegationRejectsAmbiguousLearnerReply(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	_ = workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning})
	conversation, _ := conversationstore.New("topic")
	proposal, _, _ := conversation.AppendMessage("teacher", "completed", "", []conversationstore.Block{{Type: "markdown", Source: "我准备让助教核查当前结论并给出证据。是否同意？"}})
	service := newTestService(fakeGateway{proposalID: proposal.ID})
	var accepted bool
	_ = service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_ambiguous", Content: "你看着办"}, func(frame StreamFrame) { accepted = accepted || frame.Type == "task-accepted" })
	if accepted {
		t.Fatal("ambiguous learner reply authorized delegation")
	}
	messages, err := conversation.AllMessages()
	if err != nil || len(messages) == 0 || !strings.Contains(messageText(messages[len(messages)-1]), "助教任务未创建") {
		t.Fatalf("rejection notice was not persisted: err=%v messages=%v", err, messages)
	}
}

func TestDelegationRejectsObjectiveOutsideProposal(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	_ = workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning})
	conversation, _ := conversationstore.New("topic")
	proposal, _, _ := conversation.AppendMessage("teacher", "completed", "", []conversationstore.Block{{Type: "markdown", Source: "我准备让助教制作一张概念图。是否同意？"}})
	service := newTestService(fakeGateway{proposalID: proposal.ID})
	var accepted bool
	_ = service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_changed", Content: "同意"}, func(frame StreamFrame) { accepted = accepted || frame.Type == "task-accepted" })
	if accepted {
		t.Fatal("revised objective was accepted without a new proposal")
	}
}

func TestApprovedDelegationNarrowsExpandedObjectiveToRealProposal(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	_ = workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning})
	conversation, _ := conversationstore.New("topic")
	proposalText := `可以，这次方案完整地放在本条消息里：

助教方案
- 任务内容：MapReduce 论文中文精读材料，以及 Python 集群调度模拟器和网页动画。
- 预期产出：论文要点、调度与容错设计解析、权衡分析、术语表；三种调度策略的对比数据表、图表和展示任务派发、本地命中、故障重试的动画。
- 使用资料：仅公开论文，不使用任何本地文件。
- 学习意义：把调度哲学变成看得见的数据和动画。

请回复“同意”。`
	_, _, _ = conversation.AppendMessage("teacher", "completed", "", []conversationstore.Block{{Type: "markdown", Source: proposalText}})
	expanded := `制作第一课“MapReduce 与调度哲学”的配套学习材料。第一部分基于 Dean 与 Ghemawat 2004 年公开论文，按章节整理核心要点，解析 Master 任务分配、数据本地性、推测执行、worker/master 失败处理、任务重试、combiner、本地写盘和设计权衡，并制作中英术语表。第二部分用 Python 模拟 20 台 worker、200 个三副本数据块，对比随机分配、数据本地性优先、含故障注入与推测执行三种策略，输出网络传输量、总完成时间、甘特图、柱状图和网页动画，帮助学习者直观理解调度与容错哲学。`
	service := newTestService(fakeGateway{objective: expanded})
	accepted := false
	if err := service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_real_expansion", Content: "同意"}, func(frame StreamFrame) {
		accepted = accepted || frame.Type == "task-accepted"
	}); err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("a detailed restatement of the approved MapReduce plan was rejected")
	}
	inputs := service.Authorizer.(*fakeTaskAuthorizer).inputs
	if len(inputs) != 1 || inputs[0].Objective != proposalText {
		t.Fatalf("expanded tool input widened the approved objective: %v", inputs)
	}
}

func TestDelegationRetryReusesApprovedProposalAfterTechnicalRejection(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	_ = workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning})
	conversation, _ := conversationstore.New("topic")
	proposalText := "助教方案：任务内容是制作 MapReduce 论文精读和调度实验；使用资料仅限公开论文；预期产出是中文材料、数据图表和网页动画；学习意义是直观理解调度与容错。请确认是否同意。"
	proposal, _, _ := conversation.AppendMessage("teacher", "completed", "", []conversationstore.Block{{Type: "markdown", Source: proposalText}})
	_, _, _ = conversation.AppendMessage("learner", "completed", "old-approval", []conversationstore.Block{{Type: "markdown", Source: "同意"}})
	_, _, _ = conversation.AppendMessage("teacher", "completed", "", []conversationstore.Block{{Type: "markdown", Source: "收到，正在正式提交。\n\n> 助教任务未创建。我需要重新说明方案。"}})
	service := newTestService(fakeGateway{objective: "制作 MapReduce 论文精读材料、调度实验、数据图表和网页动画"})
	accepted := false
	if err := service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_retry", Content: "重试"}, func(frame StreamFrame) {
		accepted = accepted || frame.Type == "task-accepted"
	}); err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("technical rejection incorrectly revoked the existing approval")
	}
	inputs := service.Authorizer.(*fakeTaskAuthorizer).inputs
	if len(inputs) != 1 || inputs[0].ProposalMessageID != proposal.ID {
		t.Fatalf("retry did not preserve the approved proposal: %v", inputs)
	}
}

func TestDelegationAcceptsLearnerSelectedSourceDisclosedByName(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("topic", "主题", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	sources, err := sourcestore.New("topic")
	if err != nil {
		t.Fatal(err)
	}
	source, _, err := sources.Add("闭包讲义", "notes.txt", "text/plain", 5, bytes.NewBufferString("notes"), false)
	if err != nil {
		t.Fatal(err)
	}
	conversation, _ := conversationstore.New("topic")
	_, _, _ = conversation.AppendMessage("learner", "completed", "select-source", []conversationstore.Block{{Type: "markdown", Source: "请结合这份资料"}, {Type: "attachment", ArtifactRef: source.SourceID}})
	proposal, _, _ := conversation.AppendMessage("teacher", "completed", "", []conversationstore.Block{{Type: "markdown", Source: "我准备让助教读取闭包讲义并核查当前结论，产出证据说明。是否同意？"}})
	service := newTestService(sourceGateway{proposalID: proposal.ID, sourceID: source.SourceID})
	accepted := false
	if err := service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_source", Content: "同意"}, func(frame StreamFrame) { accepted = accepted || frame.Type == "task-accepted" }); err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("learner-selected and name-disclosed source was not authorized")
	}
}

type fakeSearcher struct {
	queries []string
}

func (f *fakeSearcher) Search(_ context.Context, query string) ([]websearch.Result, error) {
	f.queries = append(f.queries, query)
	return []websearch.Result{{Title: "Go 1.24 发布说明", URL: "https://go.dev/doc/go1.24", Snippet: "泛型别名正式落地。"}}, nil
}

type searchLoopGateway struct {
	mu       sync.Mutex
	requests []teachergateway.Request
}

func (g *searchLoopGateway) record(request teachergateway.Request) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.requests = append(g.requests, request)
}

func (g *searchLoopGateway) count() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	return len(g.requests)
}

func (g *searchLoopGateway) Stream(_ context.Context, in teachergateway.Request, emit func(teachergateway.Event)) error {
	g.record(in)
	if g.count() == 1 {
		emit(teachergateway.Event{Type: "text-delta", Delta: "我先查一下。"})
		emit(teachergateway.Event{Type: "tool-call-ready", ToolCall: &teachergateway.ToolCall{CallID: "call_search", ProviderCallID: "provider_call_search", ToolName: "search_web", Arguments: map[string]any{"query": "go 1.24 发布"}}})
		return nil
	}
	emit(teachergateway.Event{Type: "text-delta", Delta: "根据检索结果回答。"})
	return nil
}

func swapSearcherForTest(t *testing.T, searcher websearch.Searcher) {
	t.Helper()
	previous := resolveSearcher
	resolveSearcher = func() websearch.Searcher { return searcher }
	t.Cleanup(func() { resolveSearcher = previous })
}

func TestSearchToolLoopFeedsResultsBack(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("searchloop", "搜索", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	gateway := &searchLoopGateway{}
	service := New(gateway)
	searcher := &fakeSearcher{}
	swapSearcherForTest(t, searcher)

	var frames []StreamFrame
	if err := service.StreamTurn(context.Background(), "searchloop", TurnInput{OperationID: "op_search", Content: "go 1.24 有什么新特性？"}, func(frame StreamFrame) {
		frames = append(frames, frame)
	}); err != nil {
		t.Fatal(err)
	}
	if gateway.count() != 2 {
		t.Fatalf("expected two provider rounds, got %d", gateway.count())
	}
	if len(searcher.queries) != 1 || searcher.queries[0] != "go 1.24 发布" {
		t.Fatalf("search query wrong: %#v", searcher.queries)
	}
	second := gateway.requests[len(gateway.requests)-1]
	if len(second.Messages) < 2 {
		t.Fatalf("follow-up messages missing: %#v", second.Messages)
	}
	assistantCall := second.Messages[len(second.Messages)-2]
	toolResult := second.Messages[len(second.Messages)-1]
	if len(assistantCall.ToolCalls) != 1 || assistantCall.ToolCalls[0].ToolName != "search_web" {
		t.Fatalf("assistant tool_calls message wrong: %#v", assistantCall)
	}
	if toolResult.ToolCallID != "provider_call_search" || !strings.Contains(toolResult.Content, "Go 1.24 发布说明") {
		t.Fatalf("tool result message wrong: %#v", toolResult)
	}
	var started, completed bool
	for _, frame := range frames {
		if frame.Type == "search-started" {
			started = true
		}
		if frame.Type == "search-completed" {
			completed = frame.Data["count"] == 1
		}
	}
	if !started || !completed {
		t.Fatalf("search frames missing: %#v", frames)
	}
}

func TestSearchLoopBoundedToTwoSearches(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("searchbound", "有界", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	// Every round emits another search call; the loop must stop after two.
	gateway := &alwaysSearchGateway{}
	service := New(gateway)
	swapSearcherForTest(t, &fakeSearcher{})
	if err := service.StreamTurn(context.Background(), "searchbound", TurnInput{OperationID: "op_bound", Content: "持续检索"}, func(StreamFrame) {}); err != nil {
		t.Fatal(err)
	}
	if rounds := gateway.count(); rounds != 3 {
		t.Fatalf("expected 3 provider rounds (2 search + 1 final), got %d", rounds)
	}
}

type alwaysSearchGateway struct {
	searchLoopGateway
}

func (g *alwaysSearchGateway) Stream(_ context.Context, in teachergateway.Request, emit func(teachergateway.Event)) error {
	g.record(in)
	emit(teachergateway.Event{Type: "text-delta", Delta: "再查一次。"})
	emit(teachergateway.Event{Type: "tool-call-ready", ToolCall: &teachergateway.ToolCall{CallID: idgen.New("call"), ProviderCallID: idgen.New("pcall"), ToolName: "search_web", Arguments: map[string]any{"query": "x"}}})
	return nil
}

// captureTextGateway records every provider request and emits one distinct
// reply per call so tests can identify which reply belongs to which turn.
type captureTextGateway struct {
	searchLoopGateway
}

func (g *captureTextGateway) Stream(_ context.Context, in teachergateway.Request, emit func(teachergateway.Event)) error {
	g.record(in)
	emit(teachergateway.Event{Type: "text-delta", Delta: "回答" + strconv.Itoa(g.count())})
	return nil
}

func TestRegenerateRemovesOldReplyFromProviderContext(t *testing.T) {
	root := t.TempDir()
	workspace.SetProjectsRootForTest(root)
	defer workspace.SetProjectsRootForTest("")
	if err := workspace.CreateProjectSkeletonWithInput("regenctx", "重生成上下文", "", workspace.ProjectInput{ProjectType: workspace.ProjectTypeSystemLearning}); err != nil {
		t.Fatal(err)
	}
	gateway := &captureTextGateway{}
	service := New(gateway)

	if err := service.StreamTurn(context.Background(), "regenctx", TurnInput{OperationID: "op_r1", Content: "第一个问题"}, func(StreamFrame) {}); err != nil {
		t.Fatal(err)
	}
	if err := service.StreamTurn(context.Background(), "regenctx", TurnInput{OperationID: "op_r2", Content: "第二个问题"}, func(StreamFrame) {}); err != nil {
		t.Fatal(err)
	}
	store, err := conversationstore.New("regenctx")
	if err != nil {
		t.Fatal(err)
	}
	latest, ok, err := store.LatestResponseID()
	if err != nil || !ok {
		t.Fatalf("no latest response: %v %v", ok, err)
	}
	var frames []StreamFrame
	if err := service.RegenerateResponse(context.Background(), "regenctx", latest, "", func(frame StreamFrame) { frames = append(frames, frame) }); err != nil {
		t.Fatal(err)
	}

	// The regeneration request must carry the full history minus the
	// superseded reply: turn-1 Q&A present, turn-2 question present,
	// turn-2's old reply ("回答2") gone.
	last := gateway.requests[len(gateway.requests)-1]
	joined, _ := json.Marshal(last.Messages)
	payload := string(joined)
	for _, needle := range []string{"第一个问题", "回答1", "第二个问题"} {
		if !strings.Contains(payload, needle) {
			t.Fatalf("regenerated context missing %q: %s", needle, payload)
		}
	}
	if strings.Contains(payload, "回答2") {
		t.Fatalf("superseded reply leaked into provider context: %s", payload)
	}
	// The fresh reply streams out of the regenerated turn.
	var streamed bool
	for _, frame := range frames {
		if frame.Type == "text-delta" && strings.Contains(frame.Data["delta"].(string), "回答3") {
			streamed = true
		}
	}
	if !streamed {
		t.Fatalf("regenerated reply was not streamed: %#v", frames)
	}
}
