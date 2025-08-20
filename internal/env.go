package internal

import (
	"fmt"
	"os"
	"strconv"
)

const AgentPortEnvName = "OP_AGENT_PORT"

func GetAgentPort() int {
	port := StandardPort

	portStr := os.Getenv(AgentPortEnvName)
	if portStr != "" {
		if parsedPort, err := strconv.Atoi(portStr); err == nil {
			port = parsedPort
		}
	}

	return port
}

const AgentHostEnvName = "OP_AGENT_HOST"

func GetAgentHost(inContainer bool) string {
	defaultHost := "localhost"

	if inContainer {
		defaultHost = "host.docker.internal"
	}

	return GetEnvOr(AgentHostEnvName, defaultHost)
}

type AgentCommand string

const (
	AgentCommandOp        AgentCommand = "op"
	AgentCommandHandshake AgentCommand = "handshake"
)

func GetAgentURL(inContainer bool, command AgentCommand) string {
	host := GetAgentHost(inContainer)
	port := GetAgentPort()

	return fmt.Sprintf("http://%s:%d/%s", host, port, command)
}

func GetEnvOr(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}
