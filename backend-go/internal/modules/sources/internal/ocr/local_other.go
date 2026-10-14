//go:build !windows && !darwin && !linux

package ocr

import (
	"context"
	"errors"
)

func localPageProvider() (PageProvider, bool) { return PageProvider{}, false }
func extractLocalPage(context.Context, []byte) (PageResult, error) {
	return PageResult{}, errors.New("windows-ocr-unavailable")
}
