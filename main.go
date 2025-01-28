//go:build darwin

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	logCommandTimeFormat = "2006-01-02 15:04:05"
)

type Config struct {
	Collectors []OSLogCollectorConfig `yaml:"collectors"`
	PIDFile    string                 `yaml:"pid_file"`
}

type OSLogCollectorConfig struct {
	Name         string `yaml:"name"`
	Predicate    string `yaml:"predicate"`
	OutputFile   string `yaml:"output_file"`
	PositionFile string `yaml:"position_file"`
	Interval     int    `yaml:"interval"`
}

type OSLogCollector struct {
	Name          string
	Predicate     string
	OutputFile    string
	PositionFile  string
	Interval      int
	LastTimestamp string

	logFile *os.File
	mu      sync.Mutex
}

type Position struct {
	LastTimestamp string `json:"last_timestamp"`
}

func loadConfig(filename string) (*Config, error) {
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

func createPIDFile(pidFile string) error {
	pid := os.Getpid()
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func removePIDFile(pidFile string) {
	os.Remove(pidFile)
}

func NewOSLogCollector(config OSLogCollectorConfig) (*OSLogCollector, error) {
	collector := &OSLogCollector{
		Name:         config.Name,
		Predicate:    config.Predicate,
		OutputFile:   config.OutputFile,
		PositionFile: config.PositionFile,
		Interval:     config.Interval,
	}

	if err := collector.loadPosition(); err != nil {
		return nil, err
	}

	if err := collector.openLogFile(); err != nil {
		return nil, err
	}

	return collector, nil
}

func (c *OSLogCollector) openLogFile() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.logFile != nil {
		c.logFile.Close()
	}

	file, err := os.OpenFile(c.OutputFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("error opening file: %v", err)
	}

	c.logFile = file
	return nil
}

func (c *OSLogCollector) writeToLogFile(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.logFile == nil {
		return fmt.Errorf("file is not open")
	}

	_, err := c.logFile.Write(data)
	if err != nil {
		return fmt.Errorf("error writing to file: %v", err)
	}
	return nil
}

func (c *OSLogCollector) loadPosition() error {
	data, err := os.ReadFile(c.PositionFile)
	if os.IsNotExist(err) {
		c.LastTimestamp = time.Now().Add(-time.Hour).Format(logCommandTimeFormat)
		return nil
	} else if err != nil {
		return fmt.Errorf("error reading position file: %v", err)
	}

	var pos Position
	if err := json.Unmarshal(data, &pos); err != nil {
		return fmt.Errorf("error parsing position file: %v", err)
	}

	c.LastTimestamp = pos.LastTimestamp
	return nil
}

func (c *OSLogCollector) savePosition() error {
	pos := Position{LastTimestamp: c.LastTimestamp}
	data, err := json.Marshal(pos)
	if err != nil {
		return fmt.Errorf("error marshaling position: %v", err)
	}

	if err := os.WriteFile(c.PositionFile, data, 0644); err != nil {
		return fmt.Errorf("error writing position file: %v", err)
	}

	return nil
}

func (c *OSLogCollector) collectLogs() error {
	for {
		endTime := time.Now().Format(logCommandTimeFormat)
		cmd := exec.Command("log", "show", "--predicate", c.Predicate, "--style", "ndjson", "--start", c.LastTimestamp, "--end", endTime)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("error executing log command: %v, output: %s", err, string(output))
		}

		if err := c.writeToLogFile(output); err != nil {
			return err
		}

		c.LastTimestamp = endTime
		if err := c.savePosition(); err != nil {
			return err
		}

		time.Sleep(time.Duration(c.Interval) * time.Second)
	}
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: ./%s <config_file>\n", os.Args[0])
	}

	configFile := os.Args[1]
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		os.Exit(1)
	}

	if err := createPIDFile(config.PIDFile); err != nil {
		log.Fatalf("Error creating PID file: %v", err)
	}
	defer removePIDFile(config.PIDFile)

	collectors := make([]*OSLogCollector, 0, len(config.Collectors))
	for i := range config.Collectors {
		collector, err := NewOSLogCollector(config.Collectors[i])
		if err != nil {
			log.Fatalf("Error creating collector: %v", err)
			os.Exit(1)
		}
		collectors = append(collectors, collector)
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGUSR1:
				// Some log rotate tools like newsyslog do not support copytruncate, so the old log file remains open.
				// When receiving the USR1 signal (30), reopen the log file to handle this.
				for _, collector := range collectors {
					if err := collector.openLogFile(); err != nil {
						log.Printf("Error opening log file for %s: %v\n", collector.Name, err)
					}
				}
			case syscall.SIGTERM, syscall.SIGINT:
				removePIDFile(config.PIDFile)
				os.Exit(0)
			}
		}
	}()

	var wg sync.WaitGroup
	for _, collector := range collectors {
		wg.Add(1)
		go func(c *OSLogCollector) {
			// TODO: Pass context to collectLogs() to make it cancelable and possibly support graceful shutdown
			defer wg.Done()
			if err := c.collectLogs(); err != nil {
				log.Fatalf("Error collecting logs for %s: %v\n", c.Name, err)
			}
		}(collector)
	}

	wg.Wait()
	removePIDFile(config.PIDFile)
}
