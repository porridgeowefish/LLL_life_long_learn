// Package sources is the public facade for source originals and revisions.
package sources

import (
	ocr "github.com/xmz14/lll/backend-go/internal/modules/sources/internal/ocr"
	impl "github.com/xmz14/lll/backend-go/internal/modules/sources/internal/store"
)

const DefaultFileLimit = impl.DefaultFileLimit
const DefaultUnitByteLimit = impl.DefaultUnitByteLimit

type Source = impl.Source
type FileRef = impl.FileRef
type Privacy = impl.Privacy
type Revision = impl.Revision
type Store = impl.Store

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
