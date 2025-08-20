package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"

	opagent "github.com/kossnocorp/op-agent"
	"github.com/kossnocorp/op-agent/internal"
	"github.com/spf13/cobra"
)

var (
	insecureMode   bool
	nonInteractive bool
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

	var approved bool
	var persistent bool

	// Approve command unless in insecure mode
	if !insecureMode {
		approved_, source, persistent_, err := approveCommand(args)
		if err != nil {
			fmt.Printf("Error checking command approval: %v\n", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		approved = approved_
		persistent = persistent_

		if logErr := internal.LogCommandRequest(args, approved_, source); logErr != nil {
			fmt.Printf("Warning: Failed to log command: %v\n", logErr)
		}
	} else {
		approved = true
		persistent = false

		if logErr := internal.LogCommandRequest(args, true, internal.ApprovalSourceInsecure); logErr != nil {
			fmt.Printf("Warning: Failed to log command: %v\n", logErr)
		}
	}

	var response internal.OpResponse

	if approved {
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

		// Save command to config only if it succeeded and was approved with "always"
		if !insecureMode && persistent && exitCode == 0 {
			config, err := internal.LoadConfig()
			if err != nil {
				fmt.Printf("Warning: Failed to load config for saving: %v\n", err)
			} else {
				config.AddApprovedCommand(args)

				if saveErr := config.SaveConfig(); saveErr != nil {
					fmt.Printf("Warning: Failed to save approved command to config: %v\n", saveErr)
				}
			}
		}

		response = internal.OpResponse{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
			Exit:   exitCode,
		}
	} else {
		response = internal.OpResponse{
			Stdout: "",
			Stderr: "The command wasn't approved by the host",
			Exit:   1,
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func approveCommand(args []string) (bool, internal.ApprovalSource, bool, error) {
	config, err := internal.LoadConfig()
	if err != nil {
		return false, internal.ApprovalSourceNonInteractive, false, fmt.Errorf("failed to load config: %v", err)
	}

	if config.IsCommandApproved(args) {
		return true, internal.ApprovalSourceConfig, false, nil
	}

	// If not interactive mode, deny commands not in config
	if nonInteractive || !internal.IsInteractive() {
		return false, internal.ApprovalSourceNonInteractive, false, nil
	}

	commandStr := strings.Join(args, " ")
	fmt.Printf("\n🔵 Command approval required:\n\n   op %s\n\n", commandStr)
	fmt.Printf("Approve? (y/o)nce, (a)lways, anything else for no: ")

	char, err := readSingleChar()
	if err != nil {
		return false, "", false, fmt.Errorf("failed to read input: %v", err)
	}
	response := strings.ToLower(string(char))

	fmt.Printf("\n")

	var approved = false
	var source = internal.ApprovalSourceInteractiveDenied
	var persistent = false

	switch response {
	case "o", "y":
		approved = true
		source = internal.ApprovalSourceInteractiveOnce
		persistent = false
	case "a":
		approved = true
		source = internal.ApprovalSourceInteractiveAlways
		persistent = true
	}

	return approved, source, persistent, nil
}

func preApproveCommand(args []string) error {
	config, err := internal.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load config: %v", err)
	}

	if config.IsCommandApproved(args) {
		fmt.Printf("Command already approved: op %s\n", strings.Join(args, " "))
		return nil
	}

	config.AddApprovedCommand(args)

	if err := config.SaveConfig(); err != nil {
		return fmt.Errorf("failed to save config: %v", err)
	}

	return nil
}

func readSingleChar() (byte, error) {
	// Try to set terminal to raw mode for immediate input
	sttyCmd := exec.Command("stty", "-icanon", "-echo", "min", "1", "time", "0")
	sttyCmd.Stdin = os.Stdin
	sttyCmd.Stdout = os.Stdout
	sttyCmd.Stderr = os.Stderr

	if err := sttyCmd.Run(); err != nil {
		// Fallback: if stty fails, just read normally
		fmt.Printf("(Press Enter after choice) ")
		reader := bufio.NewReader(os.Stdin)
		input, err := reader.ReadString('\n')
		if err != nil {
			return 0, err
		}
		input = strings.TrimSpace(input)
		if len(input) > 0 {
			return input[0], nil
		}
		return 'n', nil // Default to 'no'
	}

	// Restore normal terminal behavior on exit
	defer func() {
		restoreCmd := exec.Command("stty", "icanon", "echo")
		restoreCmd.Stdin = os.Stdin
		restoreCmd.Stdout = os.Stdout
		restoreCmd.Stderr = os.Stderr
		restoreCmd.Run()
		fmt.Printf("\n") // Add newline after character input
	}()

	// Read single character
	var char [1]byte
	n, err := os.Stdin.Read(char[:])
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 'n', nil // Default to 'no'
	}

	return char[0], nil
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
				opagent.PrintVersion()
				return
			}

			if err := startServer(); err != nil {
				fmt.Fprintf(os.Stderr, "Error starting server: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().BoolVar(&versionFlag, "version", false, "Print version information")
	rootCmd.Flags().BoolVar(&insecureMode, "insecure", false, "Disable command approval checks (UNSAFE)")
	rootCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Run in non-interactive mode (only allow pre-approved commands)")

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

	startCmd.Flags().BoolVar(&insecureMode, "insecure", false, "Disable command approval checks (UNSAFE)")
	startCmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Run in non-interactive mode (only allow pre-approved commands)")

	approveCmd := &cobra.Command{
		Use:                "approve op [command...]",
		Short:              "Pre-approve a 1Password CLI command",
		Long:               "Add a 1Password CLI command to the approved commands list without executing it.",
		DisableFlagParsing: true,
		Run: func(cmd *cobra.Command, args []string) {
			if args[0] != "op" {
				fmt.Fprintf(os.Stderr, "Error: First argument must be 'op'\n")
				fmt.Fprintf(os.Stderr, "Usage: op-agent approve op [command...]\n")
				os.Exit(1)
			}

			// Everything after 'op' is the command to approve
			opArgs := args[1:]

			if err := preApproveCommand(opArgs); err != nil {
				fmt.Fprintf(os.Stderr, "Error approving command: %v\n", err)
				os.Exit(1)
			}

			fmt.Printf("🟢 Command approved: op %s\n", strings.Join(opArgs, " "))
		},
	}

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(approveCmd)

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

	if insecureMode {
		fmt.Printf("🟡 WARNING: Running in INSECURE mode - all commands will be allowed!\n")
	}

	opPath := fmt.Sprintf("/%s", internal.AgentCommandOp)
	http.HandleFunc(opPath, handleOpCommand)

	handshakePath := fmt.Sprintf("/%s", internal.AgentCommandHandshake)
	http.HandleFunc(handshakePath, handleHandshake)

	fmt.Printf("🟣 op-agent listening on :%d\n\n", port)
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
