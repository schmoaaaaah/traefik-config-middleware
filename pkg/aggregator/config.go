package aggregator

import (
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads the application configuration from the specified YAML file.
// If poll_interval is not specified, defaults to 30s.
func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}

	if config.PollInterval == "" {
		config.PollInterval = "30s"
	}

	return &config, nil
}
