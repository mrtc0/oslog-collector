//go:build darwin

package main

import (
	"log"
	"os"

	oslog_collector "github.com/mrtc0/oslog-collector"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Usage: ./%s <config_file>\n", os.Args[0])
	}

	configFile := os.Args[1]

	agent, err := oslog_collector.NewAgentFromConfigFile(configFile)
	if err != nil {
		log.Fatalf("failed to create oslog-collector agent: %v", err)
	}

	if err := agent.Run(); err != nil {
		log.Fatalf("failed to run oslog-collector agent: %v", err)
	}
}
