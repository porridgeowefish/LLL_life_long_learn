package ocr

import (
	"context"
	"strings"
)

// ExtractChatImage never resolves a cloud binding and never creates a Source.
func ExtractChatImage(ctx context.Context, image []byte, language string) (string, error) {
	result, err := extractChatImage(ctx, image, language)
	if err != nil && (strings.Contains(err.Error(), "ocr-no-text")) {
		return "", nil
	}
	return result.Content, err
}
