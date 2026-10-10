//go:build !windows

package desktop

import "errors"

func FocusExisting(string) bool       { return false }
func ShowError(string)                {}
func RunWindow(*Engine, string) error { return errors.New("native desktop is supported on Windows") }
