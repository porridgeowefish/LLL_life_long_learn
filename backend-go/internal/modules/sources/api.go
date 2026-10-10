// Package sources is the public facade for source originals and revisions.
package sources

import (
	"context"
	ocr "github.com/xmz14/lll/backend-go/internal/modules/sources/internal/ocr"
	processing "github.com/xmz14/lll/backend-go/internal/modules/sources/internal/processing"
	retrieval "github.com/xmz14/lll/backend-go/internal/modules/sources/internal/retrieval"
	impl "github.com/xmz14/lll/backend-go/internal/modules/sources/internal/store"
)

const DefaultFileLimit = impl.DefaultFileLimit
const DefaultUnitByteLimit = impl.DefaultUnitByteLimit

var ErrSourceDeleted = impl.ErrSourceDeleted
var ErrRevisionChanged = impl.ErrRevisionChanged
var ErrSourceNotReadable = impl.ErrSourceNotReadable

type Source = impl.Source
type FileRef = impl.FileRef
type Privacy = impl.Privacy
type Revision = impl.Revision
type Store = impl.Store
type Processor = processing.Processor
type Reader = retrieval.Reader
type SourceDocument = retrieval.Document
type SourceResult = retrieval.Result
type SourceToolError = retrieval.ToolError
type SourceCitation = retrieval.Citation
type SourceRange = retrieval.Range
type SourceCoverage = retrieval.Coverage
type SourceUnitRange = retrieval.UnitRange
type SourceOutlineTarget = retrieval.Target
type ListSourceArgs = retrieval.ListArgs
type OutlineSourceArgs = retrieval.OutlineArgs
type SearchSourceArgs = retrieval.SearchArgs
type ReadSourceArgs = retrieval.ReadArgs
type SourceListData = retrieval.ListData
type SourceOutlineData = retrieval.OutlineData
type SourceSearchData = retrieval.SearchData
type SourceReadData = retrieval.ReadData
type SourceEvidencePreview = retrieval.EvidencePreview
type SourceOutlineNode = retrieval.Node
type SourceSearchHit = retrieval.Hit
type RetrievalStatus = impl.RetrievalStatus

func NewReader(store *Store) *Reader { return retrieval.New(store) }

func NewProcessor(onUpdated func(string, Source)) *Processor { return processing.New(onUpdated) }
func ProcessLocal(ctx context.Context, store *Store, sourceID string) (Source, error) {
	return processing.Process(ctx, store, sourceID)
}

func ParseDisposition(filename, mediaType string) (string, string) {
	return impl.ParseDisposition(filename, mediaType)
}

func New(slug string) (*Store, error) { return impl.New(slug) }

// ImageOCRConfigured reports whether askAiProviders.bindings.ocr resolves to
// a usable vision provider.
func ImageOCRConfigured() bool { return ocr.Configured() }

// ProcessImageOCR extracts text from an uploaded image revision through the
// configured vision model and commits it as the single derived content.md.
func ProcessImageOCR(slug string, store *Store, sourceID, revisionID string) error {
	return ocr.ProcessImageOCR(slug, store, sourceID, revisionID)
}
