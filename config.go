package oslog_collector

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	// Collectors is a list of OS Log collectors
	Collectors []OSLogCollectorConfig `yaml:"collectors"`
	// PIDFile is the file to write the process ID to
	PIDFile string `yaml:"pid_file"`
}

type OSLogCollectorConfig struct {
	// Name is the name of the collector
	Name string `yaml:"name"`
	// Predicate is the condition to get logs, which is a string passed to the --predicate option of the log command
	Predicate string `yaml:"predicate"`
	// OutputFile is the file to write the logs to
	OutputFile string `yaml:"output_file"`
	// PositionFile is the file to record the collection position of the logs
	// If this file exists, logs are collected from the position recorded in this file.
	PositionFile string `yaml:"position_file"`
	// Interval is the interval to collect logs in seconds
	Interval int `yaml:"interval"`
	// WithInfoLevel is a flag to enable the --info option of the log command
	WithInfoLevel bool `yaml:"with_info_level"`
}

func LoadConfigFromFile(filename string) (*Config, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("error reading config file: %v", err)
	}

	cfg, err := ParseConfig(data)
	if err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	return cfg, nil
}

func ParseConfig(rawConfig []byte) (*Config, error) {
	var config Config
	if err := yaml.Unmarshal(rawConfig, &config); err != nil {
		return nil, fmt.Errorf("error parsing config file: %v", err)
	}

	if err := validateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return &config, nil
}

func validateConfig(config *Config) error {
	if len(config.Collectors) == 0 {
		return fmt.Errorf("no collectors defined")
	}

	if err := validateCollectorName(config.Collectors); err != nil {
		return err
	}

	for _, c := range config.Collectors {
		if err := validateOutputFile(c.OutputFile); err != nil {
			return err
		}

		if err := validatePositionFile(c.PositionFile); err != nil {
			return err
		}

		if err := validateInterval(c.Interval); err != nil {
			return err
		}

		if err := validatePredicate(c.Predicate); err != nil {
			return err
		}
	}

	return nil
}

func validateCollectorName(collectors []OSLogCollectorConfig) error {
	names := map[string]struct{}{}
	for _, c := range collectors {
		if _, ok := names[c.Name]; ok {
			return fmt.Errorf("duplicate collector name: %s", c.Name)
		}
		names[c.Name] = struct{}{}
	}

	return nil
}

func validateOutputFile(outputFile string) error {
	if outputFile == "" {
		return fmt.Errorf("output_file is required")
	}

	return nil
}

func validatePositionFile(positionFile string) error {
	if positionFile == "" {
		return fmt.Errorf("position_file is required")
	}

	return nil
}

func validateInterval(interval int) error {
	if interval <= 0 {
		return fmt.Errorf("interval must be greater than 0")
	}

	return nil
}

func validatePredicate(predicate string) error {
	if predicate == "" {
		return fmt.Errorf("predicate is required")
	}

	return nil
}
