package oslog_collector_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Songmu/flextime"
	oslog_collector "github.com/mrtc0/oslog-collector"
	"github.com/stretchr/testify/assert"
)

type mockLogCommandRunner struct {
}

func (m *mockLogCommandRunner) RunLogCommand() ([]byte, error) {
	dummyLog := []byte("test log ")
	return dummyLog, nil
}

func TestOSLogCollector_CollectLog(t *testing.T) {
	type expect struct {
		logs string
	}

	type arrange struct {
		nowTime time.Time
	}

	testCases := map[string]struct {
		arrange
		expect
		interval time.Duration
	}{
		"when interval with 60 seconds": {
			arrange: arrange{
				nowTime: time.Date(2025, 1, 29, 0, 0, 0, 0, time.Now().Location()),
			},
			expect: expect{
				logs: "test log test log test log ", // "test log " * 3
			},
			interval: 60 * time.Second,
		},
		"when interval with 30 seconds": {
			arrange: arrange{
				nowTime: time.Date(2025, 1, 29, 0, 0, 0, 0, time.Now().Location()),
			},
			expect: expect{
				logs: "test log test log test log ", // "test log " * 3
			},
			interval: 30 * time.Second,
		},
	}

	logCollectTimes := 3

	for name, tt := range testCases {
		tt := tt

		t.Run(name, func(t *testing.T) {
			flextime.Set(tt.nowTime)

			workdir := t.TempDir()
			filename := filepath.Join(workdir, fmt.Sprintf("%d", time.Now().UnixNano()))

			cfg := oslog_collector.OSLogCollectorConfig{
				Name:         "test",
				Predicate:    "eventMessage contains[cd] \"test\"",
				OutputFile:   filename + ".log",
				PositionFile: filename + ".pos",
				Interval:     1,
			}

			count := 0

			dummyRunnerGeenerator := func(args []string) oslog_collector.LogCommandRunner {
				var startTime string
				var endTime string

				if count == 1 {
					// If the pos file does not exist at the first startup, the start time will be the current time.
					startTime = tt.nowTime.Format(oslog_collector.LogCommandTimeFormat)
					endTime = flextime.Now().Format(oslog_collector.LogCommandTimeFormat)
				} else {
					// From the second time onwards, the end time of the previous time becomes the start time.
					startTime = flextime.Now().Add(-tt.interval).Format(oslog_collector.LogCommandTimeFormat)
					endTime = flextime.Now().Format(oslog_collector.LogCommandTimeFormat)
				}

				assert.Equal(t, []string{"log", "show", "--predicate", "eventMessage contains[cd] \"test\"", "--start", startTime, "--end", endTime, "--style", "ndjson"}, args)
				return &mockLogCommandRunner{}
			}

			collector, err := oslog_collector.NewOSLogCollector(
				cfg,
				oslog_collector.WithLogCommandRunner(dummyRunnerGeenerator),
			)
			assert.NoError(t, err)

			for i := 0; i < logCollectTimes; i++ {
				count++

				t.Run(fmt.Sprintf("collect %d", i), func(t *testing.T) {
					err = collector.CollectLogs()
					assert.NoError(t, err)

					pos, err := os.ReadFile(cfg.PositionFile)
					assert.NoError(t, err)
					assert.Equal(t, fmt.Sprintf("{\"last_timestamp\":\"%s\"}", flextime.Now().Format(oslog_collector.LogCommandTimeFormat)), string(pos))
				})

				// emulate the sleep
				flextime.Set(flextime.Now().Add(tt.interval))
			}

			logs, err := os.ReadFile(cfg.OutputFile)
			assert.NoError(t, err)
			assert.Equal(t, tt.logs, string(logs))
		})
	}

	flextime.Restore()
}
