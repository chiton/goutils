package file

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/ilyakaznacheev/cleanenv"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/config"
	"gitlab.edgecastcdn.net/edgecast/web-platform/identity/goutils/log"
)

type Provider[T config.Config] struct {
	path string
}

func (provider *Provider[T]) String() string {
	builder := strings.Builder{}

	builder.WriteString(fmt.Sprintf("path: %s\n", provider.path))

	return builder.String()
}

// NewProvider creates a config provider to read configuration from file
func NewProvider[T config.Config](path string) (*Provider[T], error) {
	var zero Provider[T]

	if path == "" {
		return &zero, errors.New("path to yml file is empty")
	}

	return &Provider[T]{
		path: path,
	}, nil
}

func (cfgFile *Provider[T]) GetConfig(ctx context.Context, cfg T) error {
	logger := log.FromContext(ctx)

	logger.Infof("Loading config from file %s", cfgFile.path)
	if err := cleanenv.ReadConfig(cfgFile.path, cfg); err != nil {
		return err
	}

	return nil
}
