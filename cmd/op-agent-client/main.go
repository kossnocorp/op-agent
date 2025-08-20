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
	var versionFlag bool
	var quietFlag bool

	rootCmd := &cobra.Command{
		Use:   "op-agent-client",
		Short: "1Password CLI agent client",
		Long:  "op-agent-client connects to op-agent server to execute 1Password CLI commands.",
		Run: func(cmd *cobra.Command, args []string) {
			if versionFlag {
				version()
				return
			}
			cmd.Help()
		},
	}

	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "Print version information")
	rootCmd.PersistentFlags().BoolVarP(&quietFlag, "quiet", "q", false, "Suppress version mismatch warnings")

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			version()
		},
	}

	opCmd := &cobra.Command{
		Use:   "op [arguments...]",
		Short: "Execute op command via op-agent",
		Long:  "Execute 1Password CLI command by forwarding it to the op-agent server.",
		Args:  cobra.ArbitraryArgs,
		Run: func(cmd *cobra.Command, args []string) {
			executeOpCommand(args, quietFlag)
		},
	}

	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(opCmd)

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
