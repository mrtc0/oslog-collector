package oslog_collector

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

type Agent struct {
	Config        *Config
	LogCollectors []*OSLogCollector

	ReopenLogCh chan struct{}
	ShutdownCh  chan struct{}
}

func NewAgentFromConfigFile(configFilePath string) (*Agent, error) {
	config, err := LoadConfigFromFile(configFilePath)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}

	logCollectors, err := newOSLogCollectors(config)
	if err != nil {
		return nil, fmt.Errorf("error creating log collectors: %w", err)
	}

	return &Agent{
		Config:        config,
		LogCollectors: logCollectors,
		ReopenLogCh:   MakeReopenLogCh(),
		ShutdownCh:    MakeShutdownCh(),
	}, nil
}

func (a *Agent) Run() error {
	if err := a.storePIDFile(a.Config.PIDFile); err != nil {
		return err
	}

	defer func() {
		if err := a.removePIDFile(a.Config.PIDFile); err != nil {
			slog.Error("Error removing PID file", "error", err)
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()

		for {
			select {
			case <-a.ReopenLogCh:
				if err := a.reopenLogFiles(); err != nil {
					slog.Error("Error reopening log files", "error", err)
				}
			case <-a.ShutdownCh:
				cancel()
			case <-ctx.Done():
				return
			}
		}
	}()

	slog.Info("oslog-collector agent started")

	go StartLogCollectors(ctx, a.LogCollectors)

	wg.Wait()

	slog.Info("oslog-collector agent stopped")

	return nil
}

func (a *Agent) reopenLogFiles() error {
	for _, collector := range a.LogCollectors {
		if err := collector.OpenLogFile(); err != nil {
			return err
		}
	}
	return nil
}

func (a *Agent) storePIDFile(pidPath string) error {
	if pidPath == "" {
		return nil
	}

	pidFile, err := os.OpenFile(pidPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open pid file: %w", err)
	}
	defer pidFile.Close()

	pid := os.Getpid()
	if _, err = pidFile.WriteString(fmt.Sprintf("%d", pid)); err != nil {
		return fmt.Errorf("failed to write pid file: %w", err)
	}

	return nil
}

func (a *Agent) removePIDFile(pidPath string) error {
	if pidPath == "" {
		return nil
	}
	return os.Remove(pidPath)
}

func newOSLogCollectors(config *Config) ([]*OSLogCollector, error) {
	collectors := make([]*OSLogCollector, 0, len(config.Collectors))
	for i := range config.Collectors {
		collector, err := NewOSLogCollector(config.Collectors[i], WithLogCommandRunner(NewLogCommandRunner))
		if err != nil {
			return nil, err
		}
		collectors = append(collectors, collector)
	}
	return collectors, nil
}

// MakeShutdownCh creates a channel that will be closed when the process receives a SIGINT or SIGTERM signal.
func MakeShutdownCh() chan struct{} {
	resultCh := make(chan struct{})

	shutdownCh := make(chan os.Signal, 1)
	signal.Notify(shutdownCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-shutdownCh
		close(resultCh)
	}()

	return resultCh
}

// MakeReopenLogCh creates a channel that will receive a value when the process receives a SIGUSR1 signal.
// Some log rotate tools like newsyslog do not support copytruncate, so the old log file remains open.
// When receiving the USR1 signal (30), reopen log collector's log file to handle this.
func MakeReopenLogCh() chan struct{} {
	resultCh := make(chan struct{})

	reopenLogCh := make(chan os.Signal, 1)
	signal.Notify(reopenLogCh, syscall.SIGUSR1)

	go func() {
		for {
			<-reopenLogCh
			resultCh <- struct{}{}
		}
	}()

	return resultCh
}
