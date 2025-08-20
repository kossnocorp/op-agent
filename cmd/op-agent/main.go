package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"syscall"

	opagent "github.com/kossnocorp/op-agent"
	"github.com/kossnocorp/op-agent/internal"
	"github.com/spf13/cobra"
)

func handleOpCommand(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var args []string
	if err := json.NewDecoder(r.Body).Decode(&args); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	cmd := exec.Command("op", args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	exitCode := 0
	if err := cmd.Run(); err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.Sys().(syscall.WaitStatus).ExitStatus()
		} else {
			exitCode = 1
		}
	}

	response := internal.OpResponse{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
		Exit:   exitCode,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleHandshake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	response := internal.HandshakeResponse{
		Version: opagent.Version,
		Whoami:  "op-agent",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func main() {
	var versionFlag bool

	rootCmd := &cobra.Command{
		Use:   "op-agent",
		Short: "1Password CLI agent",
		Long: `op-agent is a CLI that allows containers to access the host
1Password CLI and its biometric authentication.`,
		Run: func(cmd *cobra.Command, args []string) {
			if versionFlag {
				version()
				return
			}

			if err := startServer(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "Print version information")

	startCmd := &cobra.Command{
		Use:   "start",
		Short: "Start the op-agent",
		Long:  "Start the op-agent server to accept client requests.",
		Run: func(cmd *cobra.Command, args []string) {
			if err := startServer(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.AddCommand(startCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func startServer() error {
	port, err := findAvailablePort(internal.StandardPort)
	if err != nil {
		return err
	}

	if port != internal.StandardPort {
		fmt.Printf("Port %d unavailable, using %d. Set %s=%d\n", internal.StandardPort, port, internal.AgentPortEnvName, port)
	}

	opPath := fmt.Sprintf("/%s", internal.AgentCommandOp)
	http.HandleFunc(opPath, handleOpCommand)

	handshakePath := fmt.Sprintf("/%s", internal.AgentCommandHandshake)
	http.HandleFunc(handshakePath, handleHandshake)

	fmt.Printf("op-agent listening on :%d\n", port)
	return http.ListenAndServe(":"+strconv.Itoa(port), nil)
}

func findAvailablePort(startPort int) (int, error) {
	for port := startPort; port < startPort+100; port++ {
		listener, err := net.Listen("tcp", ":"+strconv.Itoa(port))
		if err == nil {
			listener.Close()
			return port, nil
		}
	}
	return 0, fmt.Errorf("no available port found")
}

func version() {
	fmt.Printf("op-agent version %s\n", opagent.Version)
}
