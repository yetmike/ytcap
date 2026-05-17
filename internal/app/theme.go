package app

import (
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"gopkg.in/yaml.v3"

	"github.com/yetmike/ytcap/internal/config"
	"github.com/yetmike/ytcap/skins"
)

type Theme struct {
	Name   string           `yaml:"name"`
	Colors map[string]string `yaml:"colors"`
}

func LoadTheme(skinName string) *Theme {
	// Try user skins dir first
	userSkinPath := filepath.Join(config.DataDir(), "skins", skinName+".yaml")
	if data, err := os.ReadFile(userSkinPath); err == nil {
		var t Theme
		if yaml.Unmarshal(data, &t) == nil {
			return &t
		}
	}

	// Fall back to embedded skins
	data, err := skins.FS.ReadFile(skinName + ".yaml")
	if err != nil {
		data, _ = skins.FS.ReadFile("default.yaml")
	}

	var t Theme
	_ = yaml.Unmarshal(data, &t)
	return &t
}

func (t *Theme) Color(name string) tcell.Color {
	if hex, ok := t.Colors[name]; ok {
		return tcell.GetColor(hex)
	}
	return tcell.ColorDefault
}
