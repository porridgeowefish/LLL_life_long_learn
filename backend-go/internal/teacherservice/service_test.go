package teacherservice

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/xmz14/lll/backend-go/internal/assistanttask"
	"github.com/xmz14/lll/backend-go/internal/conversationstore"
	sourcestore "github.com/xmz14/lll/backend-go/internal/modules/sources"
	"github.com/xmz14/lll/backend-go/internal/teachergateway"
	"github.com/xmz14/lll/backend-go/internal/workspace"
)

type fakeGateway struct {
	proposalID string
	objective  string
	sourceRefs []string
}

func (f fakeGateway) Stream(_ context.Context, _ teachergateway.Request, emit func(teachergateway.Event)) error {
	objective := f.objective
	if objective == "" {
		objective = "核查当前结论"
	}
	emit(teachergateway.Event{Type: "text-delta", Delta: "助教已开始。"})
	emit(teachergateway.Event{Type: "tool-call-ready", ToolCall: &teachergateway.ToolCall{CallID: "call_test", ToolName: "delegate_learning_work", Arguments: map[string]any{"taskType": "verify", "objective": objective, "sourceRefs": f.sourceRefs, "proposalMessageId": f.proposalID}}})
	emit(teachergateway.Event{Type: "response-completed"})
	return nil
}

type sourceGateway struct {
	proposalID string
	sourceID   string
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
	service := New(fakeGateway{proposalID: proposal.ID})
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
	service := New(fakeGateway{proposalID: "msg_stale", objective: "整理 MapReduce 论文精读材料，包含调度、容错和权衡分析", sourceRefs: []string{"public:MapReduce paper"}})
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
	service := New(fakeGateway{proposalID: proposal.ID})
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
	service := New(fakeGateway{proposalID: proposal.ID})
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
	service := New(fakeGateway{objective: expanded})
	accepted := false
	if err := service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_real_expansion", Content: "同意"}, func(frame StreamFrame) {
		accepted = accepted || frame.Type == "task-accepted"
	}); err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("a detailed restatement of the approved MapReduce plan was rejected")
	}
	tasks, err := assistanttask.New("topic")
	if err != nil {
		t.Fatal(err)
	}
	items, err := tasks.List("")
	if err != nil || len(items) != 1 {
		t.Fatalf("expected one task: err=%v tasks=%v", err, items)
	}
	if items[0].Objective != proposalText {
		t.Fatalf("expanded tool input widened the approved objective: %q", items[0].Objective)
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
	service := New(fakeGateway{objective: "制作 MapReduce 论文精读材料、调度实验、数据图表和网页动画"})
	accepted := false
	if err := service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_retry", Content: "重试"}, func(frame StreamFrame) {
		accepted = accepted || frame.Type == "task-accepted"
	}); err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("technical rejection incorrectly revoked the existing approval")
	}
	store, err := assistanttask.New("topic")
	if err != nil {
		t.Fatal(err)
	}
	items, err := store.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || items[0].Origin.ProposalMessageID != proposal.ID {
		t.Fatalf("retry did not preserve the approved proposal: %v", items)
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
	service := New(sourceGateway{proposalID: proposal.ID, sourceID: source.SourceID})
	accepted := false
	if err := service.StreamTurn(context.Background(), "topic", TurnInput{OperationID: "op_source", Content: "同意"}, func(frame StreamFrame) { accepted = accepted || frame.Type == "task-accepted" }); err != nil {
		t.Fatal(err)
	}
	if !accepted {
		t.Fatal("learner-selected and name-disclosed source was not authorized")
	}
}
