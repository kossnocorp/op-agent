package opagent

import (
	_ "embed"
	"strings"
)

//go:embed VERSION
var versionFile string

// Version is the current version of op-agent.
var Version = strings.TrimSpace(versionFile)
