//go:build !windows

package desktop

import "errors"

func launchInstaller(string, string, string) error {
	return errors.New("in-app installation is available on Windows; download the new package from the release page")
}
