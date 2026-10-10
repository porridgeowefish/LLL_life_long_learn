package assistanttask

import (
	"encoding/json"
	"errors"
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
	if err = json.Unmarshal(data, &result); err != nil {
		return result, err
	}
	var raw map[string]json.RawMessage
	if err = json.Unmarshal(data, &raw); err != nil {
		return result, err
	}
	if value, ok := raw["deliverables"]; ok && string(value) != "null" && string(value) != "[]" {
		return result, errors.New("retired assistant output")
	}
	if value, ok := raw["sourceUpdate"]; ok && string(value) != "null" {
		return result, errors.New("assistant cannot publish source parsing")
	}
	for key := range result.AssetUpdates {
		if key != "review" && key != "practice" {
			return result, errors.New("unsupported asset update")
		}
	}
	return result, nil
}
