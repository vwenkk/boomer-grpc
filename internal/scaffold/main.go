package scaffold

import (
	"embed"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
)

type PluginType string

const (
	Empty  PluginType = "empty"
	Ignore PluginType = "ignore"
	Py     PluginType = "py"
	Go     PluginType = "go"
)

type ProjectInfo struct {
	ProjectName string    `json:"project_name,omitempty" yaml:"project_name,omitempty"`
	CreateTime  time.Time `json:"create_time,omitempty" yaml:"create_time,omitempty"`
	Version     string    `json:"hrp_version,omitempty" yaml:"hrp_version,omitempty"`
}

//go:embed templates/*
var templatesDir embed.FS

// CopyFile copies a file from templates dir to scaffold project
func CopyFile(templateFile, targetFile string) error {
	log.Info().Str("path", targetFile).Msg("create file")
	content, err := templatesDir.ReadFile(templateFile)
	if err != nil {
		return errors.Wrap(err, "template file not found")
	}

	err = os.WriteFile(targetFile, content, 0o644)
	if err != nil {
		log.Error().Err(err).Msg("create file failed")
		return err
	}
	return nil
}
