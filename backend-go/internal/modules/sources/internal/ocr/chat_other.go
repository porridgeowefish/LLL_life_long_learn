//go:build !windows && !linux

package ocr

import "context"

func extractChatImage(ctx context.Context, image []byte, language string) (PageResult, error) {
	return extractLocalPage(ctx, image)
}
