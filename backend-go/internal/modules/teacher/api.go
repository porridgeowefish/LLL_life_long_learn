// Package teacher is the public boundary for teaching conversations and AI providers.
package teacher

import (
	"context"

	aiconfig "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/aiconfig"
	conversation "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/conversation"
	gateway "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/gateway"
	providers "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/providers"
	service "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/service"
	usage "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/usagestore"
)

const (
	ConversationSchemaVersion = conversation.SchemaVersion
	SystemPrompt              = service.SystemPrompt
)

type (
	StreamFrame     = service.StreamFrame
	TurnInput       = service.TurnInput
	Service         = service.Service
	DelegationInput = service.DelegationInput
	DelegatedTask   = service.DelegatedTask
	TaskAuthorizer  = service.TaskAuthorizer
	ActiveResponse  = service.ActiveResponse
	ActiveResponses = service.ActiveResponses

	Block             = conversation.Block
	Message           = conversation.Message
	ConversationEvent = conversation.Event
	ConversationMeta  = conversation.Meta
	Unit              = conversation.Unit
	Projection        = conversation.Projection
	TaskLink          = conversation.TaskLink
	SequencedMessage  = conversation.SequencedMessage
	ResponseRecord    = conversation.ResponseRecord
	ResponseInfo      = conversation.ResponseInfo
	QueueItem         = conversation.QueueItem
	PromotedTurn      = conversation.PromotedTurn
	ConversationStore = conversation.Store
	UsageRecord       = usage.Record
	UsageConversation = usage.Conversation

	Gateway        = gateway.Gateway
	GatewayMessage = gateway.Message
	Tool           = gateway.Tool
	ToolCall       = gateway.ToolCall
	GatewayEvent   = gateway.Event
	GatewayRequest = gateway.Request
	Configured     = gateway.Configured

	ProviderConfig = aiconfig.Provider
	Config         = aiconfig.Config
	Binding        = aiconfig.Binding

	Provider  = providers.Provider
	AIMessage = providers.Message
	AIFrame   = providers.Frame
)

// ErrQueueItemNotFound marks queue operations against an item that already
// left the queue (promoted or discarded).
var ErrQueueItemNotFound = conversation.ErrQueueItemNotFound

func New(gateway Gateway) *Service { return service.New(gateway) }

func NewActiveResponses() *ActiveResponses { return service.NewActiveResponses() }

func NewConversation(slug string) (*ConversationStore, error) { return conversation.New(slug) }

func ListTeacherUsage() ([]UsageConversation, error) { return usage.ListAll() }

func RecordTeacherUsage(slug string, record UsageRecord) error { return usage.Append(slug, record) }

func ReconcileAllInterruptedResponses() { conversation.ReconcileAllInterruptedResponses() }

func LoadConfig() (*Config, error) { return aiconfig.Load() }

func SaveConfig(config Config) error { return aiconfig.Save(config) }

func UseConfigPathForTest(path string) func() { return aiconfig.UseConfigPathForTest(path) }

func Stream(ctx context.Context, provider Provider, system string, messages []AIMessage, onFrame func(AIFrame)) error {
	return providers.Stream(ctx, provider, system, messages, onFrame)
}

func Complete(ctx context.Context, provider Provider, system string, messages []AIMessage) (string, error) {
	return providers.Complete(ctx, provider, system, messages)
}
