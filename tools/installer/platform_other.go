//go:build !windows

package main

import "fmt"

func showMessage(title, text string, confirm bool) bool { return false }
func createIntegration(root, identity string) error {
	return fmt.Errorf("Windows installer must run on Windows")
}
func removeIntegration(identity string) error {
	return fmt.Errorf("Windows installer must run on Windows")
}
func deleteAfterExit(self, root string) error {
	return fmt.Errorf("Windows installer must run on Windows")
}
func launchApplication(root string) error { return fmt.Errorf("Windows installer must run on Windows") }
