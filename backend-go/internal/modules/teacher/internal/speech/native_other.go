//go:build !windows && !darwin

package speech

import (
	"context"
	"errors"
)

func run(context.Context, string, string, []byte) (Result, error) {
	return Result{}, errors.New("system-speech-unavailable")
}
