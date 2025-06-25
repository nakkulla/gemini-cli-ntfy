package main

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/nakkulla/gemini-cli-ntfy/pkg/config"
	flag "github.com/spf13/pflag"
)

func main() {
	// Parse our flags and separate Gemini's flags
	var (
		configPath string
		quiet      bool
		help       bool
	)

	// Manually parse arguments to separate our flags from Gemini's
	ourArgs := []string{}
	geminiArgs := []string{}

	i := 1 // Skip program name
	for i < len(os.Args) {
		arg := os.Args[i]

		// Check if it's one of our flags
		switch arg {
		case "--config", "-config":
			ourArgs = append(ourArgs, arg)
			if i+1 < len(os.Args) && !strings.HasPrefix(os.Args[i+1], "-") {
				ourArgs = append(ourArgs, os.Args[i+1])
				i++
			}
		case "--quiet", "-quiet":
			ourArgs = append(ourArgs, arg)
		case "--help", "-help":
			ourArgs = append(ourArgs, arg)
		default:
			// Handle --flag=value format for our flags
			if strings.HasPrefix(arg, "--config=") || strings.HasPrefix(arg, "-config=") {
				ourArgs = append(ourArgs, arg)
			} else {
				// Everything else goes to Gemini
				geminiArgs = append(geminiArgs, arg)
			}
		}
		i++
	}

	// Check for help flag early in original args
	for _, arg := range os.Args[1:] {
		if arg == "-help" || arg == "--help" || arg == "-h" {
			// Only show our help if no gemini args were provided
			hasGeminiArgs := false
			for _, a := range os.Args[1:] {
				if a != "-help" && a != "--help" && a != "-h" && a != "--quiet" && a != "-quiet" &&
					!strings.HasPrefix(a, "--config") && !strings.HasPrefix(a, "-config") {
					hasGeminiArgs = true
					break
				}
			}
			if !hasGeminiArgs {
				printUsage()
				os.Exit(0)
			}
		}
	}

	// Define our flags first
	flag.CommandLine.SetOutput(os.Stderr)
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.BoolVar(&quiet, "quiet", false, "Disable all notifications")
	flag.BoolVar(&help, "help", false, "Show help message")

	// Parse only our flags
	if err := flag.CommandLine.Parse(ourArgs); err != nil {
		// Don't print error for help flag
		if err != flag.ErrHelp {
			fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
			os.Exit(1)
		}
		// For help error, we've already shown our custom help above
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Override config with command line flags
	if configPath != "" {
		if err := os.Setenv("GEMINI_NOTIFY_CONFIG", configPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error setting config path: %v\n", err)
			os.Exit(1)
		}
	}
	if quiet {
		cfg.Quiet = true
	}

	// Use the manually parsed Gemini args
	userArgs := geminiArgs

	// Debug output
	if os.Getenv("GEMINI_NOTIFY_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "gemini-cli-ntfy: Parsed gemini args: %v\n", geminiArgs)
	}

	var command string

	// Determine gemini path
	if cfg.GeminiPath != "" {
		// Use configured path directly - don't validate, let it fail at execution if wrong
		command = cfg.GeminiPath
		if os.Getenv("GEMINI_NOTIFY_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "gemini-cli-ntfy: Using configured gemini path: %s\n", command)
		}
	} else {
		// Try to find gemini in PATH, excluding ourselves
		geminiPath, err := findGemini()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			fmt.Fprintf(os.Stderr, "\nYou can fix this by:\n")
			fmt.Fprintf(os.Stderr, "1. Setting gemini_path in your config file (~/.config/gemini-cli-ntfy/config.yaml)\n")
			fmt.Fprintf(os.Stderr, "2. Setting GEMINI_NOTIFY_GEMINI_PATH environment variable\n")
			fmt.Fprintf(os.Stderr, "3. Ensuring the real gemini is in your PATH\n")
			os.Exit(1)
		}
		command = geminiPath
		if os.Getenv("GEMINI_NOTIFY_DEBUG") == "1" {
			fmt.Fprintf(os.Stderr, "gemini-cli-ntfy: Found gemini in PATH at: %s\n", command)
		}
	}

	// Merge default args with user args
	var args []string
	if len(cfg.DefaultGeminiArgs) > 0 {
		args = append(args, cfg.DefaultGeminiArgs...)
	}
	args = append(args, userArgs...)

	// Create dependencies
	deps, err := NewDependencies(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating dependencies: %v\n", err)
		os.Exit(1)
	}
	defer deps.Close()

	// Create application
	app := NewApplication(deps)

	// Setup graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Ensure terminal restoration on panic
	defer func() {
		if r := recover(); r != nil {
			_ = app.Stop() // Best effort terminal restoration
			panic(r)       // Re-panic
		}
	}()

	go func() {
		<-sigChan
		// Attempt graceful shutdown
		if err := app.Stop(); err != nil {
			fmt.Fprintf(os.Stderr, "Error stopping process: %v\n", err)
		}
		// Exit with standard interrupt code
		os.Exit(130)
	}()

	// Debug output if verbose
	if os.Getenv("GEMINI_NOTIFY_DEBUG") == "1" {
		fmt.Fprintf(os.Stderr, "gemini-cli-ntfy: Starting gemini with args: %v\n", args)
		fmt.Fprintf(os.Stderr, "gemini-cli-ntfy: Config: quiet=%v, topic=%q\n", cfg.Quiet, cfg.NtfyTopic)
	}

	// Run the application
	if err := app.Run(command, args); err != nil {
		// Check if it's an exec.ExitError
		if _, ok := err.(*exec.ExitError); !ok {
			// Only log if it's not an expected exit error
			fmt.Fprintf(os.Stderr, "Error running gemini: %v\n", err)
		}
	}

	// Exit with the same code as the wrapped process
	os.Exit(app.ExitCode())
}

func printUsage() {
	fmt.Println("gemini-cli-ntfy - Gemini CLI wrapper with notifications")
	fmt.Println()
	fmt.Println("Usage: gemini-cli-ntfy [OPTIONS] [GEMINI_ARGS...]")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("      --config string   Path to config file")
	fmt.Println("      --help            Show help message")
	fmt.Println("      --quiet           Disable all notifications")
	fmt.Println()
	fmt.Println("All unknown flags are passed through to Gemini CLI")
	fmt.Println()
	fmt.Println("Environment Variables:")
	fmt.Println("  GEMINI_NOTIFY_TOPIC       Ntfy topic for notifications")
	fmt.Println("  GEMINI_NOTIFY_SERVER      Ntfy server URL (default: https://ntfy.sh)")
	fmt.Println("  GEMINI_NOTIFY_BACKSTOP_TIMEOUT  Inactivity timeout (default: 30s)")
	fmt.Println("  GEMINI_NOTIFY_QUIET       Disable notifications (true/false)")
	fmt.Println("  GEMINI_NOTIFY_STARTUP     Send startup notification (default: true)")
	fmt.Println("  GEMINI_NOTIFY_DEFAULT_ARGS  Default Gemini args (comma-separated)")
	fmt.Println("  GEMINI_NOTIFY_CONFIG      Path to config file")
	fmt.Println("  GEMINI_NOTIFY_GEMINI_PATH  Path to the real gemini binary")
	fmt.Println()
	fmt.Println("Configuration file: ~/.config/gemini-cli-ntfy/config.yaml")
}

// findGemini searches for the real gemini binary in PATH, excluding ourselves
func findGemini() (string, error) {
	// Get our own executable path to exclude it
	ourPath, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to get our executable path: %w", err)
	}
	ourPath, err = filepath.EvalSymlinks(ourPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve our executable path: %w", err)
	}

	// Search PATH for gemini
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return "", fmt.Errorf("PATH environment variable is empty")
	}

	for _, dir := range filepath.SplitList(pathEnv) {
		geminiPath := filepath.Join(dir, "gemini")

		// Check if file exists and is executable
		info, err := os.Stat(geminiPath)
		if err != nil {
			continue // Not found in this directory
		}

		if info.Mode().IsRegular() && info.Mode()&0111 != 0 {
			// Resolve symlinks to check if it's us
			resolvedPath, err := filepath.EvalSymlinks(geminiPath)
			if err != nil {
				continue
			}

			// Skip if it's our own binary
			if resolvedPath == ourPath {
				continue
			}

			// Found a different gemini binary
			return geminiPath, nil
		}
	}

	return "", fmt.Errorf("gemini not found in PATH (excluding gemini-cli-ntfy wrapper)")
}