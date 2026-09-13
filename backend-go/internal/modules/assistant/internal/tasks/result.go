package assistanttask

import (
	"encoding/json"
	"os"
)

// readResult only decodes the assistant's publication declaration. The
// assistant owns output acceptance; the dispatcher owns publishing it.
func readResult(path string) (resultManifest, error) {
	var result resultManifest
	data, err := os.ReadFile(path)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(data, &result)
	return result, err
}
