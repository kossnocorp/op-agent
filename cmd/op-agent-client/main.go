package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	opagent "github.com/kossnocorp/op-agent"
	"github.com/kossnocorp/op-agent/internal"
	"github.com/spf13/cobra"
)

func executeOpCommand(args []string, quiet bool) {
	if err := checkHandshake(quiet); err != nil {
		fmt.Fprintf(os.Stderr, "Handshake failed: %v\n", err)
		os.Exit(1)
	}

	jsonData, err := json.Marshal(args)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error encoding arguments: %v\n", err)
		os.Exit(1)
	}

	url := internal.GetAgentURL(inContainer(), internal.AgentCommandOp)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to op-agent at %s: %v\n", url, err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Agent returned error %d: %s\n", resp.StatusCode, string(body))
		os.Exit(1)
	}

	var opResp internal.OpResponse
	if err := json.Unmarshal(body, &opResp); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing response: %v\n", err)
		os.Exit(1)
	}

	if opResp.Stdout != "" {
		fmt.Print(opResp.Stdout)
	}

	if opResp.Stderr != "" {
		fmt.Fprint(os.Stderr, opResp.Stderr)
	}

	os.Exit(opResp.Exit)
}

func main() {
	var quietFlag bool

	rootCmd := &cobra.Command{
		Use:                "op-agent-client [flags] op [command...]",
		Short:              "1Password CLI agent client",
		Long:               "op-agent-client connects to op-agent server to execute 1Password CLI commands.",
		DisableFlagParsing: true, // Parse flags manually to avoid conflicts with 'op' command flags
		Run: func(cmd *cobra.Command, args []string) {
			// Find the position of 'op' command
			opIndex := -1
			for i, arg := range args {
				if arg == "op" {
					opIndex = i
					break
				}
			}
			
			// If no 'op' found, show help
			if opIndex == -1 {
				cmd.Help()
				return
			}
			
			// Parse client flags that appear BEFORE 'op'
			clientArgs := args[:opIndex]
			opArgs := args[opIndex+1:] // Everything after 'op'
			
			// Handle client flags (only those before 'op')
			for _, arg := range clientArgs {
				switch arg {
				case "--version":
					version()
					return
				case "-q", "--quiet":
					quietFlag = true
				case "-h", "--help":
					cmd.Help()
					return
				}
			}
			
			// Execute the op command with all arguments after 'op'
			executeOpCommand(opArgs, quietFlag)
		},
	}

	// No subcommands needed since we handle everything in the root command

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func checkHandshake(quiet bool) error {
	resp, err := http.Get(internal.GetAgentURL(inContainer(), internal.AgentCommandHandshake))

	if err != nil {
		return fmt.Errorf("failed to connect to op-agent: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("handshake failed with status %d", resp.StatusCode)
	}

	var handshake internal.HandshakeResponse
	if err := json.NewDecoder(resp.Body).Decode(&handshake); err != nil {
		return fmt.Errorf("invalid handshake response: %v", err)
	}

	if handshake.Whoami != "op-agent" {
		return fmt.Errorf("unexpected server identity: %s", handshake.Whoami)
	}

	if handshake.Version != opagent.Version && !quiet {
		fmt.Fprintf(os.Stderr, "Warning: version mismatch - client: %s, server: %s\n", opagent.Version, handshake.Version)
	}

	return nil
}

func inContainer() bool {
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}
	if _, err := os.Stat("/run/.containerenv"); err == nil {
		return true
	}
	return false
}

func version() {
	fmt.Printf("op-agent-client version %s\n", opagent.Version)
}
