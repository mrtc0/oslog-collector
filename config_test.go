package oslog_collector_test

import (
	"testing"

	oslog_collector "github.com/mrtc0/oslog-collector"
	"github.com/stretchr/testify/assert"
)

func TestParseConfig(t *testing.T) {
	t.Parallel()

	testCases := map[string]struct {
		config           string
		expectErr        bool
		expectErrMessage string
	}{
		"when config is valid": {
			config:    validConfig,
			expectErr: false,
		},
		"when config has duplicate collector name": {
			config:           duplicateCollectorNameConfig,
			expectErr:        true,
			expectErrMessage: "duplicate collector name: foo",
		},
	}

	for name, tt := range testCases {
		tt := tt

		t.Run(name, func(t *testing.T) {
			_, err := oslog_collector.ParseConfig([]byte(tt.config))
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectErrMessage)
				return
			}

			assert.NoError(t, err)
		})
	}
}

var (
	validConfig = `
pid_file: /var/run/oslog-collector.pid
collectors:
  - name: foo
    output_file: /var/log/foo.log
    position_file: /var/lib/oslog-collector/foo.pos
    interval: 60
    predicate: "process == 'foo'"
    with_info_level: true
  - name: bar
    output_file: /var/log/bar.log
    position_file: /var/lib/oslog-collector/bar.pos
    interval: 60
    predicate: "process == 'bar'"
`

	duplicateCollectorNameConfig = `
pid_file: /var/run/oslog-collector.pid
collectors:
  - name: foo
    output_file: /var/log/foo.log
    position_file: /var/lib/oslog-collector/foo.pos
    interval: 60
    predicate: "process == 'foo'"
    with_info_level: true
  - name: foo
    output_file: /var/log/bar.log
    position_file: /var/lib/oslog-collector/bar.pos
    interval: 60
    predicate: "process == 'bar'"
    with_info_level: true
`
)
