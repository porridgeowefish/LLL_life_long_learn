// Package integration contains cross-module adapters and no business rules.
package integration

import (
	assistant "github.com/xmz14/lll/backend-go/internal/modules/assistant"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher"
)

type TeacherTaskAuthorizer struct{}

func (TeacherTaskAuthorizer) CreateDelegation(projectSlug string, input teacher.DelegationInput) (teacher.DelegatedTask, bool, error) {
	store, err := assistant.NewTaskStore(projectSlug)
	if err != nil {
		return teacher.DelegatedTask{}, false, err
	}
	task, created, err := store.Create(assistant.CreateInput{
		Type:       input.TaskType,
		Objective:  input.Objective,
		SourceRefs: input.SourceRefs,
		Origin: assistant.Origin{
			Kind:              "teacher-tool",
			OperationID:       input.OperationID,
			ProposalMessageID: input.ProposalMessageID,
			ApprovalMessageID: input.ApprovalMessageID,
			ToolCallID:        input.ToolCallID,
		},
		ConversationCutoffSeq: input.ConversationCutoffSeq,
	})
	return teacher.DelegatedTask{ID: task.ID, Status: task.Status}, created, err
}
