package startup

import (
	"github.com/Dicklesworthstone/ntm/internal/config"
	"github.com/Dicklesworthstone/ntm/internal/profiler"
)

// configLoader manages lazy config loading
var configLoader = NewLazy[*config.Config]("config", func() (*config.Config, error) {
	span := profiler.StartWithPhase("config_load_inner", "deferred")
	defer span.End()

	cfg, err := config.Load(configFilePath)
	if err != nil {
		// Use defaults if config doesn't exist
		return config.Default(), nil
	}
	return cfg, nil
})

// configFilePath stores the custom config path if specified
var configFilePath string

// SetConfigPath sets the config file path for lazy loading
func SetConfigPath(path string) {
	configFilePath = path
}

// GetConfig returns the configuration, loading it lazily if needed
func GetConfig() (*config.Config, error) {
	return configLoader.Get()
}

// MustGetConfig returns the configuration, panicking on error
func MustGetConfig() *config.Config {
	return configLoader.MustGet()
}

// IsConfigLoaded returns true if config has been loaded
func IsConfigLoaded() bool {
	return configLoader.IsInitialized()
}

// ResetConfig allows re-loading config (useful for testing)
func ResetConfig() {
	configLoader.Reset()
}
