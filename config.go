package oslog_collector

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Collectors []OSLogCollectorConfig `yaml:"collectors"`
	PIDFile    string                 `yaml:"pid_file"`
}

type OSLogCollectorConfig struct {
	Name          string `yaml:"name"`
	Predicate     string `yaml:"predicate"`
	OutputFile    string `yaml:"output_file"`
	PositionFile  string `yaml:"position_file"`
	Interval      int    `yaml:"interval"`
	WithInfoLevel bool   `yaml:"with_info_level"`
}

func LoadConfig(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return &config, nil
}
