package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type AutoNotesSettings struct {
	Enabled    bool   `json:"enabled"`
	EveryTurns int    `json:"everyTurns"`
	ProviderID string `json:"providerId"`
	Model      string `json:"model"`
}

func LoadTeacherSettings() (AutoNotesSettings, error) {
	out := AutoNotesSettings{Enabled: true, EveryTurns: 3}
	data, err := os.ReadFile(activePath(""))
	if errors.Is(err, os.ErrNotExist) {
		return out, nil
	}
	if err != nil {
		return out, err
	}
	var raw struct {
		Teacher struct {
			AutoNotes struct {
				Enabled    *bool  `json:"enabled"`
				EveryTurns *int   `json:"everyTurns"`
				ProviderID string `json:"providerId"`
				Model      string `json:"model"`
			} `json:"autoNotes"`
		} `json:"teacher"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return out, err
	}
	cfg := raw.Teacher.AutoNotes
	if cfg.Enabled != nil {
		out.Enabled = *cfg.Enabled
	}
	if cfg.EveryTurns != nil {
		out.EveryTurns = *cfg.EveryTurns
	}
	out.ProviderID, out.Model = cfg.ProviderID, cfg.Model
	return out, ValidateAutoNotes(out)
}
func ValidateAutoNotes(cfg AutoNotesSettings) error {
	if cfg.Model != "" && cfg.ProviderID == "" {
		return errors.New("teacher.autoNotes.model requires providerId")
	}
	if cfg.EveryTurns < 1 || cfg.EveryTurns > 100 {
		return fmt.Errorf("teacher.autoNotes.everyTurns must be 1–100")
	}
	return nil
}
func SaveTeacherSettings(cfg AutoNotesSettings) error {
	saveMu.Lock()
	defer saveMu.Unlock()
	if err := ValidateAutoNotes(cfg); err != nil {
		return err
	}
	raw, err := readRaw(activePath(""))
	if err != nil {
		return err
	}
	section, _ := raw["teacher"].(map[string]any)
	if section == nil {
		section = map[string]any{}
	}
	auto, _ := section["autoNotes"].(map[string]any)
	if auto == nil {
		auto = map[string]any{}
	}
	auto["enabled"], auto["everyTurns"], auto["providerId"], auto["model"] = cfg.Enabled, cfg.EveryTurns, cfg.ProviderID, cfg.Model
	section["autoNotes"] = auto
	raw["teacher"] = section
	return writeRaw(activePath(""), raw)
}
