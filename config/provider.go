package config

import "context"

type Provider[T Config] interface {
	// The input parameter, cfg T, must be a non-nil pointer to a config struct.
	GetConfig(ctx context.Context, cfg T) error
}
