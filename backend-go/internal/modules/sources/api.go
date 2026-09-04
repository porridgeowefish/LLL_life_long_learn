// Package sources is the public facade for source originals and revisions.
package sources

import impl "github.com/xmz14/lll/backend-go/internal/modules/sources/internal/store"

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
