package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// Command approval.
type ApprovalSource string

const (
	ApprovalSourceConfig            ApprovalSource = "config"
	ApprovalSourceInteractiveOnce   ApprovalSource = "interactive-once"
	ApprovalSourceInteractiveAlways ApprovalSource = "interactive-always"
	ApprovalSourceInteractiveDenied ApprovalSource = "interactive-denied"
	ApprovalSourceNonInteractive    ApprovalSource = "non-interactive"
	ApprovalSourceInsecure          ApprovalSource = "insecure"
)

// Server configuration stored in ~/.config/op-agent/config.json (or `%APPDATA%/op-agent/config.json` in Windows)
// NOTE: We use JSON instead of TOML/YAML to avoid additional dependencies and reduce attack surface.
type Config struct {
	ApprovedCommands []string `json:"approved"`
}

// Command request log entry.
type CommandRequestLogEntry struct {
	Timestamp string         `json:"timestamp"`
	Args      []string       `json:"args"`
	Approved  bool           `json:"approved"`
	Source    ApprovalSource `json:"source"`
}

// Command log entry.
type CommandLogEntry struct {
	Timestamp string   `json:"timestamp"`
	Args      []string `json:"args"`
	Exit      int      `json:"exit"`
}

func GetConfigDir() (string, error) {
	var configDir string

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		configDir = filepath.Join(appData, "op-agent")
	} else {
		// macOS and Linux
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME environment variable not set")
		}
		configDir = filepath.Join(home, ".config", "op-agent")
	}

	// Ensure directory exists
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create config directory: %v", err)
	}

	return configDir, nil
}

func GetLogDir() (string, error) {
	var logDir string

	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA environment variable not set")
		}
		logDir = filepath.Join(appData, "op-agent")
	} else {
		// macOS and Linux
		home := os.Getenv("HOME")
		if home == "" {
			return "", fmt.Errorf("HOME environment variable not set")
		}
		logDir = filepath.Join(home, ".local", "share", "op-agent")
	}

	// Ensure directory exists
	if err := os.MkdirAll(logDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create log directory: %v", err)
	}

	return logDir, nil
}

func LoadConfig() (*Config, error) {
	configDir, err := GetConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "config.json")
	config := &Config{
		ApprovedCommands: []string{},
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Config file doesn't exist, return empty config
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %v", err)
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %v", err)
	}

	return config, nil
}

func (c *Config) SaveConfig() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	configPath := filepath.Join(configDir, "config.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file: %v", err)
	}

	return nil
}

func (c *Config) IsCommandApproved(args []string) bool {
	commandStr := strings.Join(args, " ")
	for _, approved := range c.ApprovedCommands {
		if approved == commandStr {
			return true
		}
	}
	return false
}

func (c *Config) AddApprovedCommand(args []string) {
	commandStr := strings.Join(args, " ")
	if !c.IsCommandApproved(args) {
		c.ApprovedCommands = append(c.ApprovedCommands, commandStr)
	}
}

func LogCommandRequest(args []string, approved bool, source ApprovalSource) error {
	logEntry := CommandRequestLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Args:      args,
		Approved:  approved,
		Source:    source,
	}

	approvedStr := "🔴 Denied:"
	if approved {
		approvedStr = fmt.Sprintf("🟢 Approved via %s:", logEntry.Source)
	}
	fmt.Printf("[%s] %s op %s\n", logEntry.Timestamp, approvedStr, strings.Join(logEntry.Args, " "))

	logEntryBytes, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %v", err)
	}

	err = LogEntry(logEntryBytes)
	if err != nil {
		return err
	}

	return nil
}

func LogCommand(args []string, exit int) error {
	logEntry := CommandLogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Args:      args,
		Exit:      exit,
	}

	fmt.Printf("[%s] EXECUTED (exit code %d) op %s\n", logEntry.Timestamp, logEntry.Exit, strings.Join(logEntry.Args, " "))

	logEntryBytes, err := json.Marshal(logEntry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %v", err)
	}

	err = LogEntry(logEntryBytes)
	if err != nil {
		return err
	}

	return nil
}

func LogEntry(entry []byte) error {
	logPath, err := PrepareLog()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}
	defer file.Close()

	if _, err := fmt.Fprintf(file, "%s\n", entry); err != nil {
		return fmt.Errorf("failed to write log entry: %v", err)
	}

	return nil
}

func PrepareLog() (string, error) {
	logDir, err := GetLogDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(logDir, "commands.log"), nil
}

func IsInteractive() bool {
	// Check common environment variables that indicate non-interactive mode
	nonInteractiveVars := []string{
		"CI", "CONTINUOUS_INTEGRATION", "BUILD_ID", "BUILD_NUMBER",
		"GITHUB_ACTIONS", "GITLAB_CI", "JENKINS_URL", "TEAMCITY_VERSION",
		"TF_BUILD", "BUILDKITE", "CIRCLECI", "TRAVIS", "DRONE",
	}

	for _, envVar := range nonInteractiveVars {
		if os.Getenv(envVar) != "" {
			return false
		}
	}

	// Also check if stdin is available for interactive input
	if stat, err := os.Stdin.Stat(); err == nil {
		return stat.Mode()&os.ModeCharDevice != 0
	}

	return true
}
