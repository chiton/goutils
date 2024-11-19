package config

type Config interface {
	// Strings returns a []string representation of all the configs in the Config
	// mainly used for structured logging
	Strings() []string
	Validate() error
}
