package opagent

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version is the current version of op-agent.
var Version = strings.TrimSpace(versionFile)

func PrintVersion() {
	fmt.Printf("%s\n", Version)
}
