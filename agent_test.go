package oslog_collector_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	oslog_collector "github.com/mrtc0/oslog-collector"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgent_Run(t *testing.T) {
	t.Run("when received SIGTERM signal then shutdown", func(t *testing.T) {
		workdir := t.TempDir()
		filename := filepath.Join(workdir, fmt.Sprintf("%d", time.Now().UnixNano()))

		agent := newDummyAgent(t, filename)

		go func() {
			time.Sleep(3 * time.Second)
			agent.ShutdownCh <- struct{}{}
		}()

		err := agent.Run()
		require.NoError(t, err)
	})

	t.Run("when received SIGUSR1 signal then reopen log", func(t *testing.T) {
		workdir := t.TempDir()
		filename := filepath.Join(workdir, fmt.Sprintf("%d", time.Now().UnixNano()))

		agent := newDummyAgent(t, filename)

		go func() {
			time.Sleep(3 * time.Second)
			os.Remove(filename + ".log")

			agent.ReopenLogCh <- struct{}{}

			time.Sleep(3 * time.Second)

			_, err := os.Stat(filename + ".log")

			assert.NoError(t, err)

			agent.ShutdownCh <- struct{}{}
		}()

		err := agent.Run()
		require.NoError(t, err)
	})
}

func newDummyAgent(t *testing.T, baseFilename string) *oslog_collector.Agent {
	t.Helper()

	dummyRunnerGeenerator := func(args []string) oslog_collector.LogCommandRunner {
		return &mockLogCommandRunner{}
	}

	collectorCfg := oslog_collector.OSLogCollectorConfig{
		Name:         "test",
		Predicate:    "eventMessage contains[cd] \"test\"",
		OutputFile:   baseFilename + ".log",
		PositionFile: baseFilename + ".pos",
		Interval:     3,
	}

	cfg := &oslog_collector.Config{
		PIDFile: baseFilename + ".pid",
		Collectors: []oslog_collector.OSLogCollectorConfig{
			collectorCfg,
		},
	}

	collector, err := oslog_collector.NewOSLogCollector(collectorCfg, oslog_collector.WithLogCommandRunner(dummyRunnerGeenerator))
	require.NoError(t, err)

	agent := &oslog_collector.Agent{
		Config: cfg,
		LogCollectors: []*oslog_collector.OSLogCollector{
			collector,
		},
		ReopenLogCh: oslog_collector.MakeReopenLogCh(),
		ShutdownCh:  oslog_collector.MakeShutdownCh(),
	}

	return agent
}
