//go:build darwin

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
	config, err := oslog_collector.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
		os.Exit(1)
	}

	if err := createPIDFile(config.PIDFile); err != nil {
		log.Fatalf("Error creating PID file: %v", err)
	}
	defer removePIDFile(config.PIDFile)

	collectors := make([]*oslog_collector.OSLogCollector, 0, len(config.Collectors))
	for i := range config.Collectors {
		collector, err := oslog_collector.NewOSLogCollector(config.Collectors[i], oslog_collector.WithLogCommandRunner(oslog_collector.NewLogCommandRunner))
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
					if err := collector.OpenLogFile(); err != nil {
						log.Printf("Error opening log file for %s: %v\n", collector.Name, err)
					}
				}
			case syscall.SIGTERM, syscall.SIGINT:
				removePIDFile(config.PIDFile)
				os.Exit(0)
			}
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())

	cleanup := func() {
		cancel()
		removePIDFile(config.PIDFile)
	}
	defer cleanup()

	var wg sync.WaitGroup
	for _, collector := range collectors {
		wg.Add(1)

		go func(c *oslog_collector.OSLogCollector) {
			defer wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				default:
					if err := c.CollectLogs(); err != nil {
						log.Fatalf("Error collecting logs for %s: %v\n", c.Name, err)
					}
					time.Sleep(time.Duration(c.Interval) * time.Second)
				}
			}
		}(collector)
	}

	wg.Wait()
}
