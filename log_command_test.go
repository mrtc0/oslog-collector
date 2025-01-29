package oslog_collector_test

import (
	"testing"

	oslog_collector "github.com/mrtc0/oslog-collector"
	"github.com/stretchr/testify/assert"
)

func TestLogCommandRunner_RunLogCommand(t *testing.T) {
	t.Parallel()

}

func TestNewLogCommandBuilder(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		buildResult   []string
		expectCommand []string
	}{
		"simple": {
			buildResult:   oslog_collector.NewLogCommandBuilder().Build(),
			expectCommand: []string{"log", "show"},
		},
		"with predicate": {
			buildResult:   oslog_collector.NewLogCommandBuilder().WithPredicate("subsystem == 'com.apple.mdns'").Build(),
			expectCommand: []string{"log", "show", "--predicate", "subsystem == 'com.apple.mdns'"},
		},
		"with predicate and level": {
			buildResult:   oslog_collector.NewLogCommandBuilder().WithPredicate("subsystem == 'com.apple.mdns'").WithInfoLevel(true).Build(),
			expectCommand: []string{"log", "show", "--predicate", "subsystem == 'com.apple.mdns'", "--info"},
		},
	}

	for name, tt := range testCases {
		tt := tt

		t.Run(name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tt.buildResult, tt.expectCommand)
		})
	}
}
