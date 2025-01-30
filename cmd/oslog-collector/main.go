//go:build darwin

package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	oslog_collector "github.com/mrtc0/oslog-collector"
)

func createPIDFile(pidFile string) error {
	pid := os.Getpid()
	return os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func removePIDFile(pidFile string) {
	os.Remove(pidFile)
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: ./%s <config_file>\n", os.Args[0])
	}

	configFile := os.Args[1]
	config, err := oslog_collector.LoadConfigFromFile(configFile)
	if err != nil {
		slog.Error("Error loading config", "error", err)
		os.Exit(1)
	}

	collectors, err := newOSLogCollectors(config)
	if err != nil {
		slog.Error("Error creating log collectors", "error", err)
		os.Exit(1)
	}

	if err := createPIDFile(config.PIDFile); err != nil {
		slog.Error("Error creating PID file", "error", err)
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())

	cleanup := func() {
		removePIDFile(config.PIDFile)
	}
	defer cleanup()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGUSR1, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		for sig := range sigChan {
			switch sig {
			case syscall.SIGUSR1:
				// Some log rotate tools like newsyslog do not support copytruncate, so the old log file remains open.
				// When receiving the USR1 signal (30), reopen the log file to handle this.
				if err := reopenLogFile(collectors); err != nil {
					slog.Error("Error reopening log file", "error", err)
				}
			case syscall.SIGTERM, syscall.SIGINT:
				cancel()
			}
		}
	}()

	if err := oslog_collector.StartLogCollectors(ctx, collectors); err != nil {
		slog.Error("Error starting log collectors", "error", err)
	}
}

func newOSLogCollectors(config *oslog_collector.Config) ([]*oslog_collector.OSLogCollector, error) {
	collectors := make([]*oslog_collector.OSLogCollector, 0, len(config.Collectors))
	for i := range config.Collectors {
		collector, err := oslog_collector.NewOSLogCollector(config.Collectors[i], oslog_collector.WithLogCommandRunner(oslog_collector.NewLogCommandRunner))
		if err != nil {
			return nil, err
		}
		collectors = append(collectors, collector)
	}
	return collectors, nil
}

func reopenLogFile(collectors []*oslog_collector.OSLogCollector) error {
	for _, collector := range collectors {
		if err := collector.OpenLogFile(); err != nil {
			return err
		}
	}

	return nil
}
