package oslog_collector

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/Songmu/flextime"
)

var (
	logCommandTimeFormat = "2006-01-02 15:04:05"
	defaultStyle         = "ndjson"
)

type OSLogCollector struct {
	Name          string
	Predicate     string
	OutputFile    string
	PositionFile  string
	Interval      int
	LastTimestamp string
	WithInfoLevel bool

	logCommandRunnerGenerator LogCommandRunnerGenerator
	logFile                   *os.File
	mu                        sync.Mutex
}

type OSLogCollectorOption func(*OSLogCollector)

func WithLogCommandRunner(generator LogCommandRunnerGenerator) OSLogCollectorOption {
	return func(c *OSLogCollector) {
		c.logCommandRunnerGenerator = generator
	}
}

func NewOSLogCollector(config OSLogCollectorConfig, opts ...OSLogCollectorOption) (*OSLogCollector, error) {
	collector := &OSLogCollector{
		Name:                      config.Name,
		Predicate:                 config.Predicate,
		OutputFile:                config.OutputFile,
		PositionFile:              config.PositionFile,
		Interval:                  config.Interval,
		WithInfoLevel:             config.WithInfoLevel,
		logCommandRunnerGenerator: NewLogCommandRunner,
	}

	for _, opt := range opts {
		opt(collector)
	}

	if err := collector.loadPosition(); err != nil {
		return nil, err
	}

	if err := collector.OpenLogFile(); err != nil {
		return nil, err
	}

	return collector, nil
}

// StartLogCollectors starts the OS Log collectors in the background.
// It will run until the context is canceled.
func StartLogCollectors(ctx context.Context, collectors []*OSLogCollector) error {
	var wg sync.WaitGroup
	for _, collector := range collectors {
		wg.Add(1)

		go func(c *OSLogCollector) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if err := c.CollectLogs(); err != nil {
						slog.Error("Error collecting logs", "collector_name", c.Name, "error", err)
					}
					time.Sleep(time.Duration(c.Interval) * time.Second)
				}
			}
		}(collector)
	}

	wg.Wait()

	return nil
}

func (c *OSLogCollector) CollectLogs() error {
	endTime := flextime.Now().Format(logCommandTimeFormat)

	command := NewLogCommandBuilder().
		WithPredicate(c.Predicate).WithStartTime(c.LastTimestamp).WithEndTime(endTime).
		WithStyle(defaultStyle).WithInfoLevel(c.WithInfoLevel).
		Build()
	output, err := c.logCommandRunnerGenerator(command).RunLogCommand()
	if err != nil {
		return fmt.Errorf("error executing log command: %v, output: %s", err, string(output))
	}

	if err := c.writeToLogFile(output); err != nil {
		return err
	}

	c.LastTimestamp = endTime
	return c.savePosition()
}

func (c *OSLogCollector) OpenLogFile() error {
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
		c.LastTimestamp = flextime.Now().Format(logCommandTimeFormat)
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
