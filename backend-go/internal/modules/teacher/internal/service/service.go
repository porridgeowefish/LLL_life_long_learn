package teacherservice

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xmz14/lll/backend-go/internal/idgen"
	sourcestore "github.com/xmz14/lll/backend-go/internal/modules/sources"
	conversationstore "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/conversation"
	teachergateway "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/gateway"
	usagestore "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/usagestore"
	websearch "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/websearch"
)

const SystemPrompt = `你是 LLL 的教师。你的首要职责是与学习者进行清晰、耐心、有针对性的教学对话，而不是生产文件或代替 IDE。

你可以自然地使用五种教学动作，但它们是柔性的教学方法，不是必须逐项展示的流程：
- 引导：帮助学习者形成问题、方向和动机；
- 澄清：解释概念、边界、误解和例子；
- 验证：检验理解或主张是否成立；
- 沉淀：把已经讨论清楚的知识整理为可复用资产；
- 复盘：回看学习路径、薄弱点和下一步。

组织回答时，对于总体架构、系统主线、复杂关系或方案说明，可以参考金字塔原理：先给结论与主干，再分层给出支撑和细节。它是一种帮助理解的方法，不要机械套用固定格式。

需要图示时，禁止使用 ASCII 或其他纯文本字符画。简单的结构、关系和流程图使用 Mermaid；复杂、需要精细布局或更强视觉表达的图使用 SVG。图示应当服务于解释，并在正文中保留必要说明。
Mermaid 必须使用带 mermaid 语言标记的代码围栏。优先使用 flowchart TD 或 flowchart LR；节点 ID 只用简短 ASCII 字母数字，中文标签统一写成 A["中文标签"]；每条连线单独一行。不要在标签中嵌入 Markdown、HTML、主题初始化指令、click、classDef 或外部链接。拿不准语法时改用安全的内联 SVG，不要猜测 Mermaid 语法。

只有重活才交给助教，例如深入核查、资料解析、运行实验、信息调研、作图、生成报告或更新教学资产。你唯一可用的工具是 delegate_learning_work。

调用助教必须分成两个对话回合：
1. 先用普通教师消息说明准备交给助教的工作、会使用哪些已授权资料、预期产出和学习意义，并明确询问学习者是否同意；本回合不得调用工具。
2. 只有学习者后续明确同意，才调用工具；服务会自动把批准绑定到紧邻的上一条教师方案。

如果工具因为技术或授权实现问题返回“助教任务未创建”，这不会撤销学习者已经给出的同意。学习者随后明确要求“重试”“再来”或“继续提交”且任务范围未改变时，应直接再次调用工具，不得复述同一方案或索要第三次确认。

学习者消息中的资料附件只提供资料名称、逻辑 ID 与状态，不会把资料正文自动放入你的同步上下文。若要交给助教读取，方案中必须用学习者看得懂的资料名称明确披露；工具参数 sourceRefs 再使用对应逻辑 ID。未被学习者附件明确选择的资料不得委派。

内部消息标识绝不能出现在面向学习者的回答中，不要输出 messageId、responseId 或 msg_ 一类内部字段。
不要把路径、命令、CLI、并发、模型或实现细节作为工具参数。不要声称自己已经运行代码、读取未授权文件或完成助教工作。助教异步执行；工具接受后，继续简短教学即可。一次教师响应最多调用一次工具。`

type StreamFrame struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data"`
}

type TurnInput struct {
	OperationID    string
	Content        string
	AttachmentRefs []string
	ProviderID     string
}

type DelegationInput struct {
	TaskType              string
	Objective             string
	SourceRefs            []string
	PracticeRequested     bool
	OperationID           string
	ProposalMessageID     string
	ApprovalMessageID     string
	ToolCallID            string
	ConversationCutoffSeq uint64
}

type DelegatedTask struct {
	ID     string
	Status string
}

type TaskAuthorizer interface {
	CreateDelegation(projectSlug string, input DelegationInput) (DelegatedTask, bool, error)
}

type taskAuthorizationError interface {
	error
	DelegationCode() string
	ExistingTaskIDValue() string
}

type Service struct {
	Gateway    teachergateway.Gateway
	Authorizer TaskAuthorizer
	OnTask     func(projectSlug string, task DelegatedTask)
}

// resolveSearcher is a seam for tests; production always resolves from config.
var resolveSearcher = websearch.Configured

func New(gateway teachergateway.Gateway) *Service {
	if gateway == nil {
		gateway = teachergateway.Configured{}
	}
	return &Service{Gateway: gateway}
}

func (s *Service) StreamTurn(ctx context.Context, slug string, in TurnInput, emit func(StreamFrame)) error {
	in.Content = strings.TrimSpace(in.Content)
	if in.OperationID == "" || in.Content == "" || len([]byte(in.Content)) > 64<<10 {
		return errors.New("invalid learner turn")
	}
	conversation, err := conversationstore.New(slug)
	if err != nil {
		return err
	}
	learnerBlocks := []conversationstore.Block{{Type: "markdown", Source: in.Content}}
	for _, ref := range in.AttachmentRefs {
		learnerBlocks = append(learnerBlocks, conversationstore.Block{Type: "attachment", ArtifactRef: ref})
	}
	learner, learnerEvent, err := conversation.AppendMessage("learner", "completed", in.OperationID, learnerBlocks)
	if err != nil {
		return err
	}
	if prior, found, err := conversation.ResponseForLearner(learner.ID); err != nil {
		return err
	} else if found {
		emit(StreamFrame{Type: "turn-accepted", Data: map[string]any{"learnerMessageId": learner.ID, "responseId": prior.ResponseID}})
		if prior.Active || prior.Message == nil {
			return errors.New("teacher response is already active")
		}
		emit(StreamFrame{Type: "message-started", Data: map[string]any{"teacherMessageId": prior.TeacherMessageID}})
		for _, block := range prior.Message.Blocks {
			switch block.Type {
			case "markdown":
				emit(StreamFrame{Type: "text-delta", Data: map[string]any{"blockId": block.ID, "delta": block.Source}})
			case "reasoning-summary":
				emit(StreamFrame{Type: "reasoning-summary-delta", Data: map[string]any{"blockId": block.ID, "delta": block.Source}})
			}
		}
		meta, _ := conversation.Meta()
		emit(StreamFrame{Type: "message-completed", Data: map[string]any{"messageId": prior.TeacherMessageID, "latestSeq": meta.LatestSeq}})
		return nil
	}
	return s.streamTeacherResponse(ctx, slug, conversation, learner, learnerEvent.Seq, idgen.New("resp"), idgen.New("msg"), in.ProviderID, "", emit)
}

// StreamPromotedTurn runs a turn for a queue item that was already recorded as
// a learner message by the store's promotion step. steering=true adds the
// mid-generation guidance appendix for interrupt-and-redirect turns.
func (s *Service) StreamPromotedTurn(ctx context.Context, slug string, promoted conversationstore.PromotedTurn, steering bool, emit func(StreamFrame)) error {
	conversation, err := conversationstore.New(slug)
	if err != nil {
		return err
	}
	responseID, teacherMessageID := idgen.New("resp"), idgen.New("msg")
	steerNote := ""
	if steering {
		steerNote = messageText(promoted.Learner)
		_, _ = conversation.Append("steering-note", map[string]any{"learnerMessageId": promoted.Learner.ID, "responseId": responseID})
	}
	return s.streamTeacherResponse(ctx, slug, conversation, promoted.Learner, promoted.LearnerSeq, responseID, teacherMessageID, promoted.Item.ProviderID, steerNote, emit)
}

// RegenerateResponse supersedes an old response and immediately re-runs the
// same triggering learner message. The old reply stays in the event log but is
// hidden from every projection.
func (s *Service) RegenerateResponse(ctx context.Context, slug, oldResponseID, providerID string, emit func(StreamFrame)) error {
	conversation, err := conversationstore.New(slug)
	if err != nil {
		return err
	}
	info, found, err := conversation.ResponseByID(oldResponseID)
	if err != nil || !found || info.Superseded || info.Active {
		return errors.New("response cannot be regenerated")
	}
	if latest, ok, latestErr := conversation.LatestResponseID(); latestErr != nil || !ok || latest != oldResponseID {
		return errors.New("response cannot be regenerated: not the latest response")
	}
	learner, foundLearner, err := conversation.FindMessage(info.TriggeringLearnerMessageID)
	if err != nil || !foundLearner || learner.Role != "learner" {
		return errors.New("triggering learner message missing")
	}
	var learnerSeq uint64
	sequenced, err := conversation.SequencedMessages()
	if err != nil {
		return err
	}
	for _, item := range sequenced {
		if item.Message.ID == learner.ID {
			learnerSeq = item.Seq
			break
		}
	}
	responseID, teacherMessageID := idgen.New("resp"), idgen.New("msg")
	if err := conversation.SupersedeResponse(oldResponseID, responseID); err != nil {
		return err
	}
	return s.streamTeacherResponse(ctx, slug, conversation, learner, learnerSeq, responseID, teacherMessageID, providerID, "", emit)
}

// streamTeacherResponse is the shared streaming core for every turn origin:
// direct send, queued promotion, steering, and regeneration.
func (s *Service) streamTeacherResponse(ctx context.Context, slug string, conversation *conversationstore.Store, learner conversationstore.Message, learnerSeq uint64, responseID, teacherMessageID, providerID, steerNote string, emit func(StreamFrame)) error {
	learnerText := messageText(learner)
	textBlockID, reasoningBlockID := idgen.New("blk"), idgen.New("blk")
	emit(StreamFrame{Type: "turn-accepted", Data: map[string]any{"learnerMessageId": learner.ID, "responseId": responseID}})
	if _, err := conversation.Append("teacher-response-started", map[string]any{"responseId": responseID, "teacherMessageId": teacherMessageID, "triggeringLearnerMessageId": learner.ID}); err != nil {
		return err
	}
	emit(StreamFrame{Type: "message-started", Data: map[string]any{"teacherMessageId": teacherMessageID}})

	systemPrompt, providerMessages, err := buildTeacherContext(ctx, slug, conversation)
	if err != nil {
		message := conversationstore.Message{ID: teacherMessageID, Role: "teacher", Status: "failed", Blocks: []conversationstore.Block{}, CreatedAt: time.Now().UTC(), CompletedAt: time.Now().UTC()}
		_, _ = conversation.RecordMessage(message)
		_, _ = conversation.Append("teacher-response-finished", map[string]any{"responseId": responseID, "messageId": teacherMessageID, "status": "failed"})
		emit(StreamFrame{Type: "message-failed", Data: map[string]any{"messageId": teacherMessageID, "code": "context_preparation_failed", "partialPreserved": false}})
		return err
	}
	if isExplicitApproval(learnerText) {
		systemPrompt += "\n\n【当前回合工具门禁】当前学习者消息包含明确同意或明确要求重试。若最近一项尚未成功创建的助教方案已经完整说明内容、资料、产出和学习意义，可立即调用 delegate_learning_work；中间的技术拒绝提示不会撤销同意，无需复述方案或再次索要确认。若任务范围已经变化，才重新说明新方案。"
	} else {
		systemPrompt += "\n\n【当前回合工具门禁】当前学习者消息不是对紧邻助教方案的明确同意，本回合严禁调用 delegate_learning_work。若仍有待办助教工作，请重新完整说明方案并询问是否同意。"
	}
	if steerNote != "" {
		systemPrompt += "\n\n【生成中引导】学习者在你上一条回复未完成时插入了：" + steerNote + "\n上一条 interrupted 消息是中断稿：吸收其中仍然有效的部分，按引导方向继续教学。"
	}

	tool := teachergateway.Tool{Name: "delegate_learning_work", Description: "在教师已经向学习者披露助教方案、且学习者在后续消息中明确同意后，创建一个异步助教任务。", Schema: map[string]any{
		"type": "object", "additionalProperties": false,
		"properties": map[string]any{
			"taskType":          map[string]any{"type": "string", "enum": []string{"consolidate", "verify", "produce-material"}},
			"objective":         map[string]any{"type": "string", "maxLength": 8192},
			"sourceRefs":        map[string]any{"type": "array", "description": "仅填写学习者在消息中明确选择的本地资料 sourceId。公开网页、论文标题和普通名称不要填写；没有本地资料时必须传空数组。", "items": map[string]any{"type": "string", "pattern": "^source_"}},
			"practiceRequested": map[string]any{"type": "boolean", "description": "仅当 taskType=consolidate 且教师已在经学习者同意的方案中明确承诺同时出题时为 true；其他情况必须为 false。"},
		},
		"required": []string{"taskType", "objective", "sourceRefs"},
	}}
	tools := []teachergateway.Tool{tool}
	searcher := resolveSearcher()
	if searcher != nil {
		tools = append(tools, teachergateway.Tool{Name: "search_web", Description: "检索公开网络信息。适用于时效性强或需要事实核对的问题；搜索结果是外部信息，回答中需辨析可靠性并给出来源链接。", Schema: map[string]any{
			"type": "object", "additionalProperties": false,
			"properties": map[string]any{
				"query": map[string]any{"type": "string", "maxLength": 256, "description": "面向搜索引擎的检索词，用最可能命中的自然语言关键词。"},
			},
			"required": []string{"query"},
		}})
		systemPrompt += "\n\n你可以使用 search_web 工具检索公开网络信息（一次回答最多调用 2 次）。仅在需要时效性信息或事实核对时使用；搜索结果来自外部站点，需要辨析可靠性与时效，并在回答中给出来源链接。"
	}

	var text, reasoning strings.Builder
	usage := map[string]int{}
	acceptedTool := false
	// The search loop: stream once; if the model calls search_web and the
	// round budget allows it, answer the call and stream again with the
	// results appended. Two search rounds maximum keeps the turn bounded.
	const maxSearchRounds = 2
	var providerErr error
	roundMessages := providerMessages
	for round := 0; ; round++ {
		roundTextStart := text.Len()
		var pendingSearch *teachergateway.ToolCall
		providerErr = s.Gateway.Stream(ctx, teachergateway.Request{System: systemPrompt, Messages: roundMessages, Tools: tools, ProviderID: providerID}, func(event teachergateway.Event) {
			switch event.Type {
			case "text-delta":
				text.WriteString(event.Delta)
				emit(StreamFrame{Type: "text-delta", Data: map[string]any{"blockId": textBlockID, "delta": event.Delta}})
			case "reasoning-summary-delta":
				reasoning.WriteString(event.Delta)
				emit(StreamFrame{Type: "reasoning-summary-delta", Data: map[string]any{"blockId": reasoningBlockID, "delta": event.Delta}})
			case "usage":
				for key, value := range event.Usage {
					usage[key] = value
				}
				emit(StreamFrame{Type: "usage", Data: map[string]any{"usage": event.Usage}})
			case "tool-call-ready":
				if event.ToolCall == nil {
					return
				}
				if event.ToolCall.ToolName == "search_web" && searcher != nil && pendingSearch == nil {
					pendingSearch = event.ToolCall
					_, _ = conversation.Append("tool-call-requested", map[string]any{"callId": event.ToolCall.CallID, "responseId": responseID, "toolName": event.ToolCall.ToolName, "arguments": event.ToolCall.Arguments})
					return
				}
				_, _ = conversation.Append("tool-call-requested", map[string]any{"callId": event.ToolCall.CallID, "responseId": responseID, "toolName": event.ToolCall.ToolName, "arguments": event.ToolCall.Arguments})
				if acceptedTool {
					_, _ = conversation.Append("tool-result-recorded", map[string]any{"callId": event.ToolCall.CallID, "status": "rejected", "code": "one_task_per_response"})
					emit(StreamFrame{Type: "tool-rejected", Data: map[string]any{"toolCallId": event.ToolCall.CallID, "code": "one_task_per_response"}})
					appendToolNotice(&text, textBlockID, "one_task_per_response", emit)
					return
				}
				task, ok, toolErr := s.acceptDelegation(slug, conversation, learner, learnerSeq, learner.OperationID, *event.ToolCall)
				if toolErr != nil {
					data := map[string]any{"toolCallId": event.ToolCall.CallID, "code": delegationCode(toolErr)}
					var classified taskAuthorizationError
					if errors.As(toolErr, &classified) && classified.ExistingTaskIDValue() != "" {
						data["existingTaskId"] = classified.ExistingTaskIDValue()
					}
					emit(StreamFrame{Type: "tool-rejected", Data: data})
					appendToolNotice(&text, textBlockID, data["code"].(string), emit)
					_, _ = conversation.Append("tool-result-recorded", map[string]any{"callId": event.ToolCall.CallID, "status": "rejected", "code": data["code"]})
					return
				}
				acceptedTool = true
				_, _ = conversation.Append("task-linked", conversationstore.TaskLink{MessageID: teacherMessageID, TaskID: task.ID, ToolCallID: event.ToolCall.CallID})
				_, _ = conversation.Append("tool-result-recorded", map[string]any{"callId": event.ToolCall.CallID, "status": "accepted", "taskId": task.ID, "taskStatus": task.Status})
				emit(StreamFrame{Type: "task-accepted", Data: map[string]any{"toolCallId": event.ToolCall.CallID, "taskId": task.ID, "status": task.Status}})
				if ok && s.OnTask != nil {
					s.OnTask(slug, task)
				}
			}
		})
		if providerErr != nil || pendingSearch == nil || acceptedTool || round >= maxSearchRounds {
			break
		}
		query, _ := pendingSearch.Arguments["query"].(string)
		if strings.TrimSpace(query) == "" {
			query = learnerText
		}
		emit(StreamFrame{Type: "search-started", Data: map[string]any{"query": query}})
		resultText := ""
		results, searchErr := searcher.Search(ctx, query)
		if searchErr != nil {
			emit(StreamFrame{Type: "search-failed", Data: map[string]any{"query": query, "code": "websearch_unavailable"}})
			resultText = "搜索服务暂时不可用，请基于已有知识回答并说明未能检索。"
			_, _ = conversation.Append("tool-result-recorded", map[string]any{"callId": pendingSearch.CallID, "status": "failed", "code": "websearch_unavailable"})
		} else {
			emit(StreamFrame{Type: "search-completed", Data: map[string]any{"query": query, "count": len(results)}})
			resultText = websearch.FormatResults(results)
			_, _ = conversation.Append("tool-result-recorded", map[string]any{"callId": pendingSearch.CallID, "status": "executed", "resultCount": len(results)})
		}
		roundMessages = append(roundMessages,
			teachergateway.Message{Role: "assistant", Content: text.String()[roundTextStart:], ToolCalls: []teachergateway.ToolCall{*pendingSearch}},
			teachergateway.Message{Role: "tool", ToolCallID: pendingSearch.ProviderCallID, Content: resultText},
		)
	}

	status := "completed"
	if providerErr != nil {
		if errors.Is(providerErr, context.Canceled) {
			status = "interrupted"
		} else {
			status = "failed"
		}
	}
	var blocks []conversationstore.Block
	if reasoning.Len() > 0 {
		blocks = append(blocks, conversationstore.Block{ID: reasoningBlockID, Type: "reasoning-summary", Source: reasoning.String()})
	}
	if text.Len() > 0 {
		blocks = append(blocks, conversationstore.Block{ID: textBlockID, Type: "markdown", Source: text.String()})
	}
	message := conversationstore.Message{ID: teacherMessageID, Role: "teacher", Status: status, Blocks: blocks, CreatedAt: time.Now().UTC(), CompletedAt: time.Now().UTC()}
	finalEvent, persistErr := conversation.RecordMessage(message)
	if persistErr != nil {
		return persistErr
	}
	_, _ = conversation.Append("teacher-response-finished", map[string]any{"responseId": responseID, "messageId": teacherMessageID, "status": status})
	if meta, metaErr := conversation.Meta(); metaErr == nil {
		if unit, unitErr := conversation.Unit(); unitErr == nil {
			_ = usagestore.Append(slug, usagestore.Record{ConversationID: meta.ID, UnitID: unit.UnitID, ResponseID: responseID, ProviderID: providerID, InputTokens: usage["inputTokens"], OutputTokens: usage["outputTokens"]})
		}
	}
	if providerErr != nil {
		emit(StreamFrame{Type: "message-failed", Data: map[string]any{"messageId": teacherMessageID, "code": providerFailureCode(providerErr), "partialPreserved": len(blocks) > 0}})
		return providerErr
	}
	emit(StreamFrame{Type: "message-completed", Data: map[string]any{"messageId": teacherMessageID, "latestSeq": finalEvent.Seq}})
	return nil
}

func appendToolNotice(text *strings.Builder, blockID, code string, emit func(StreamFrame)) {
	notice := "\n\n> 助教任务未创建。"
	switch code {
	case "delegation_not_approved":
		notice += "我需要重新说明本次任务的内容、资料、预期产出和学习意义；你在下一条消息明确同意后，我再提交。"
	case "same_type_active":
		notice += "同类型工作已经在执行，本次没有重复创建。"
	case "one_task_per_response":
		notice += "一次教师回复最多创建一项助教工作。"
	default:
		notice += "任务服务暂时无法接收，请稍后重新说明方案。"
	}
	text.WriteString(notice)
	emit(StreamFrame{Type: "text-delta", Data: map[string]any{"blockId": blockID, "delta": notice}})
}

func (s *Service) acceptDelegation(slug string, conversation *conversationstore.Store, approval conversationstore.Message, cutoff uint64, operationID string, call teachergateway.ToolCall) (DelegatedTask, bool, error) {
	if call.ToolName != "delegate_learning_work" {
		return DelegatedTask{}, false, errors.New("unsupported tool")
	}
	taskType, _ := call.Arguments["taskType"].(string)
	objective, _ := call.Arguments["objective"].(string)
	practiceRequested, _ := call.Arguments["practiceRequested"].(bool)
	if taskType != "consolidate" {
		practiceRequested = false
	}
	if !isExplicitApproval(messageText(approval)) {
		return DelegatedTask{}, false, errors.New("delegation not approved")
	}
	messages, err := conversation.AllMessages()
	if err != nil {
		return DelegatedTask{}, false, err
	}
	approvalIndex := -1
	for i := range messages {
		if messages[i].ID == approval.ID {
			approvalIndex = i
			break
		}
	}
	proposal, authorizedObjective, found := findAuthorizedProposal(messages, approvalIndex, objective)
	if !found {
		return DelegatedTask{}, false, errors.New("delegation not approved: no approved proposal")
	}
	proposalID := proposal.ID
	// proposalMessageId was part of an early tool contract and leaked into some
	// persisted teacher prose. It is no longer model-owned input: old providers
	// may still echo that stale value, but authorization is derived exclusively
	// from the durable adjacent teacher/learner messages above.
	var refs []string
	if raw, ok := call.Arguments["sourceRefs"].([]any); ok {
		for _, value := range raw {
			if ref, ok := value.(string); ok && strings.HasPrefix(ref, "source_") {
				refs = append(refs, ref)
			}
		}
	}
	if raw, ok := call.Arguments["sourceRefs"].([]string); ok {
		for _, ref := range raw {
			if strings.HasPrefix(ref, "source_") {
				refs = append(refs, ref)
			}
		}
	}
	proposalText := messageText(proposal)
	sources, err := sourcestore.New(slug)
	if err != nil {
		return DelegatedTask{}, false, err
	}
	authorized := map[string]bool{}
	for i := 0; i <= approvalIndex; i++ {
		for _, block := range messages[i].Blocks {
			if block.Type == "attachment" && block.ArtifactRef != "" {
				authorized[block.ArtifactRef] = true
			}
		}
	}
	for _, ref := range refs {
		source, _, err := sources.Get(ref)
		if err != nil {
			return DelegatedTask{}, false, fmt.Errorf("invalid source ref: %w", err)
		}
		if !authorized[ref] {
			return DelegatedTask{}, false, errors.New("delegation not approved: source was not selected by learner")
		}
		if !strings.Contains(proposalText, ref) && !strings.Contains(proposalText, source.DisplayName) {
			return DelegatedTask{}, false, errors.New("delegation not approved: source was not disclosed")
		}
	}
	if s.Authorizer == nil {
		return DelegatedTask{}, false, errors.New("task authorizer unavailable")
	}
	task, created, err := s.Authorizer.CreateDelegation(slug, DelegationInput{TaskType: taskType, Objective: authorizedObjective, SourceRefs: refs, PracticeRequested: practiceRequested, OperationID: operationID + ":delegate", ProposalMessageID: proposalID, ApprovalMessageID: approval.ID, ToolCallID: call.CallID, ConversationCutoffSeq: cutoff})
	return task, created, err
}

func messageText(message conversationstore.Message) string {
	var out strings.Builder
	for _, block := range message.Blocks {
		if block.Type == "markdown" {
			out.WriteString(block.Source)
			out.WriteByte('\n')
		}
	}
	return strings.TrimSpace(out.String())
}

func isExplicitApproval(text string) bool {
	normalized := strings.ToLower(strings.TrimSpace(text))
	for _, negative := range []string{"不同意", "不可以", "不要", "别执行", "别开始", "取消", "拒绝", "do not", "don't", "no,"} {
		if strings.Contains(normalized, negative) {
			return false
		}
	}
	for _, positive := range []string{"同意", "确认", "可以", "开始吧", "执行吧", "交给助教", "允许", "批准", "重试", "再来", "继续提交", "approved", "go ahead", "retry", "yes"} {
		if strings.Contains(normalized, positive) {
			return true
		}
	}
	// Common short confirmations are approvals only as standalone words or at
	// the beginning of the reply. Avoid broad substring matching (for example,
	// "好像" and "运行" must not authorize work).
	for _, positive := range []string{"ok", "okay", "好的", "好", "行"} {
		if normalized == positive || strings.HasPrefix(normalized, positive+"，") || strings.HasPrefix(normalized, positive+",") || strings.HasPrefix(normalized, positive+"。") || strings.HasPrefix(normalized, positive+" ") {
			return true
		}
	}
	return false
}

func findAuthorizedProposal(messages []conversationstore.Message, approvalIndex int, objective string) (conversationstore.Message, string, bool) {
	for i, scanned := approvalIndex-1, 0; i >= 0 && scanned < 8; i, scanned = i-1, scanned+1 {
		message, text := messages[i], messageText(messages[i])
		switch message.Role {
		case "teacher":
			if authorized, ok := objectiveWithinProposal(text, objective); ok {
				return message, authorized, true
			}
			// A failed submission is transport history, not a new proposal and
			// not a revocation of the learner's prior approval.
			if strings.Contains(text, "助教任务未创建") {
				continue
			}
			return conversationstore.Message{}, "", false
		case "learner":
			if isExplicitApproval(text) {
				continue
			}
			return conversationstore.Message{}, "", false
		default:
			return conversationstore.Message{}, "", false
		}
	}
	return conversationstore.Message{}, "", false
}

// objectiveWithinProposal keeps the learner-approved proposal authoritative.
// A concise tool objective may be used as-is. If the provider expands it with
// extra details, the task is safely narrowed back to the approved proposal
// instead of entering an impossible propose/approve/reject loop.
func objectiveWithinProposal(proposal, objective string) (string, bool) {
	left, right := compactScopeText(proposal), compactScopeText(objective)
	if len([]rune(right)) < 3 || left == "" {
		return "", false
	}
	if strings.Contains(left, right) {
		return objective, true
	}
	if strings.Contains(right, left) {
		return proposal, true
	}
	proposalPairs, objectivePairs := runePairs(left), runePairs(right)
	if len(proposalPairs) == 0 || len(objectivePairs) == 0 {
		return "", false
	}
	matched := 0
	for pair := range objectivePairs {
		if proposalPairs[pair] {
			matched++
		}
	}
	objectiveCoverage := float64(matched) / float64(len(objectivePairs))
	if objectiveCoverage >= 0.35 {
		return objective, true
	}
	proposalCoverage := float64(matched) / float64(len(proposalPairs))
	if proposalCoverage >= 0.35 {
		return proposal, true
	}
	if completeDelegationProposal(proposal) {
		return proposal, true
	}
	return "", false
}

func completeDelegationProposal(text string) bool {
	normalized := strings.ToLower(text)
	groups := [][]string{
		{"任务内容", "工作内容", "task content", "objective"},
		{"使用资料", "资料范围", "材料范围", "sources", "materials"},
		{"预期产出", "交付成果", "deliverables", "outputs"},
		{"学习意义", "学习价值", "learning value", "significance"},
	}
	for _, group := range groups {
		found := false
		for _, marker := range group {
			if strings.Contains(normalized, marker) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func compactScopeText(text string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r >= '\u4e00' && r <= '\u9fff':
			return r
		case r >= 'A' && r <= 'Z':
			return r + ('a' - 'A')
		default:
			return -1
		}
	}, text)
}

func runePairs(text string) map[string]bool {
	runes := []rune(text)
	pairs := map[string]bool{}
	for i := 0; i+1 < len(runes); i++ {
		pairs[string(runes[i:i+2])] = true
	}
	return pairs
}

func delegationCode(err error) string {
	var classified taskAuthorizationError
	if errors.As(err, &classified) {
		return classified.DelegationCode()
	}
	if strings.Contains(err.Error(), "approved") {
		return "delegation_not_approved"
	}
	return "task_persistence_failed"
}

func providerFailureCode(err error) string {
	if errors.Is(err, context.Canceled) {
		return "interrupted"
	}
	return "teacher_provider_unavailable"
}
