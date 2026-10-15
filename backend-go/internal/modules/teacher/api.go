// Package teacher is the public boundary for teaching conversations and AI providers.
package teacher

import (
	"context"
	"errors"
	"github.com/xmz14/lll/backend-go/internal/modules/sources"
	noteservice "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/noteservice"
	notestore "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/notestore"
	"github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/speech"
	config "github.com/xmz14/lll/backend-go/internal/platform/config"
	"strings"

	aiconfig "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/aiconfig"
	conversation "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/conversation"
	gateway "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/gateway"
	providers "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/providers"
	service "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/service"
	usage "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/usagestore"
	websearch "github.com/xmz14/lll/backend-go/internal/modules/teacher/internal/websearch"
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
	ReadingCoverage   = conversation.ReadingCoverage
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

const (
	FeynmanFinishRequest = service.FeynmanFinishRequest
	FeynmanOpening       = service.FeynmanOpening
)

func FeynmanSystemPrompt(review string) string { return service.FeynmanSystemPrompt(review) }

func CheckGenerationRequest(req GatewayRequest, p ProviderConfig) (int, error) {
	return service.CheckGenerationRequest(req, p)
}
func SourceTools() []Tool { return service.GenerationSourceTools() }

func GenerationOutputTokens(p ProviderConfig) int   { return service.GenerationOutputTokens(p) }
func GenerationThinkingTokens(p ProviderConfig) int { return service.GenerationThinkingTokens(p) }

type WebSearcher = websearch.Searcher
type WebSearchResult = websearch.Result

func ConfiguredWebSearch() WebSearcher { return websearch.Configured() }

func NewActiveResponses() *ActiveResponses { return service.NewActiveResponses() }

func NewConversation(slug string) (*ConversationStore, error) { return conversation.New(slug) }

type ChatAttachment = conversation.Attachment
type SpeechCapability = speech.Capability

const MaxChatAttachmentBytes = conversation.MaxAttachmentBytes

func LocalSpeechCapability(ctx context.Context) SpeechCapability { return speech.Probe(ctx) }
func ParseChatAttachment(ctx context.Context, slug, name, language string, data []byte) (ChatAttachment, error) {
	st, err := conversation.New(slug)
	if err != nil {
		return ChatAttachment{}, err
	}
	return st.AddAttachment(ctx, name, language, data, func(ctx context.Context, kind, language string, data []byte) (string, error) {
		if kind == "audio" {
			return speech.Recognize(ctx, language, data)
		}
		text, err := sources.ExtractChatImage(ctx, data, language)
		if err != nil {
			if strings.Contains(err.Error(), "limit") {
				return "", errors.New("图片尺寸超过系统OCR限制，请缩小图片后重试")
			}
			return "", errors.New("系统文字提取不可用，请安装中文或英文OCR语言包后重试")
		}
		return text, nil
	})
}

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

type AutoNotesSettings = config.AutoNotesSettings
type NoteTurn = conversation.NoteTurn
type NoteUsage = notestore.NoteUsage
type NoteEntry = notestore.NoteEntry
type NoteJob = notestore.NoteJob
type NotesResource = notestore.NotesResource
type NoteGenerator = notestore.NoteGenerator
type NoteStore = notestore.Store
type NoteService = noteservice.Service

var ErrNoteJobNotFound = notestore.ErrNoteJobNotFound
var ErrNoteJobRunning = notestore.ErrNoteJobRunning

func NewNoteStore(slug string) (*NoteStore, error)        { return notestore.New(slug) }
func NewNoteService(generator NoteGenerator) *NoteService { return noteservice.New(generator) }
func LoadTeacherSettings() (AutoNotesSettings, error)     { return config.LoadTeacherSettings() }
func SaveTeacherSettings(cfg AutoNotesSettings) error     { return config.SaveTeacherSettings(cfg) }

type ProjectUsage = usage.ProjectUsage

func ListProjectUsage() ([]ProjectUsage, error) { return usage.ListProjects() }
