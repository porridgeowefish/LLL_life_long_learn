package assistanttask

type AssetSnapshot struct {
	VersionID          string
	ConversationCursor uint64
	Content            string
}

type AssetCommitInput struct {
	Key               string
	BaseVersionID     string
	BaseContent       string
	CandidateContent  string
	TaskID            string
	RunID             string
	FromSeq           uint64
	ThroughSeq        uint64
	SourceRevisionIDs []string
	ChangeSummary     string
}

type SourceSnapshot struct {
	RevisionID string
	SHA256     string
}

type Dependencies struct {
	ConversationSnapshot func(projectSlug string, throughSeq uint64) ([]byte, error)
	PreferencesSnapshot  func() (filename string, content []byte, err error)
	AssetKeys            func() []string
	ReadAsset            func(projectSlug, key string) (AssetSnapshot, error)
	AdvanceAsset         func(projectSlug, key string, throughSeq uint64) error
	CommitAsset          func(projectSlug string, input AssetCommitInput) (status string, eventPayload any, err error)
	SealSource           func(projectSlug, sourceID, destination string) (SourceSnapshot, error)
	CommitSourceDerived  func(projectSlug, sourceID, revisionID, taskID string, files map[string][]byte, media map[string]string) (eventPayload any, err error)
	OnPublished          func(task Task)
}
