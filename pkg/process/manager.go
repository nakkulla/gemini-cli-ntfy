package process

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/nakkulla/gemini-cli-ntfy/pkg/config"
	"github.com/nakkulla/gemini-cli-ntfy/pkg/interfaces"
)

// Manager manages the wrapped Gemini CLI process
type Manager struct {
	config        *config.Config
	ptyManager    PTY
	outputHandler interfaces.DataHandler
	inputHandler  func()
	exitCode      int
	mu            sync.Mutex
	sigChan       chan os.Signal
	done          chan struct{}
}

// NewManager creates a new process manager
func NewManager(cfg *config.Config, outputHandler interfaces.DataHandler, inputHandler func()) *Manager {
	return &Manager{
		config:        cfg,
		ptyManager:    NewPTYManager(),
		outputHandler: outputHandler,
		inputHandler:  inputHandler,
		done:          make(chan struct{}),
	}
}

// Start starts the Gemini CLI process
func (m *Manager) Start(command string, args []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for self-wrap
	if os.Getenv("GEMINI_CLI_NTFY_WRAPPED") == "1" {
		return fmt.Errorf("already wrapped by gemini-cli-ntfy")
	}

	// Set environment to prevent self-wrap
	// Create a copy of the environment and add our variable
	env := make([]string, 0, len(os.Environ())+1)
	for _, e := range os.Environ() {
		// Skip if already set to avoid duplication
		if !strings.HasPrefix(e, "GEMINI_CLI_NTFY_WRAPPED=") {
			env = append(env, e)
		}
	}
	env = append(env, "GEMINI_CLI_NTFY_WRAPPED=1")

	// Start the process with PTY
	if err := m.ptyManager.Start(command, args, env); err != nil {
		return fmt.Errorf("failed to start process: %w", err)
	}

	// Start I/O copying with output handling
	go func() {
		var handler func([]byte)
		if m.outputHandler != nil {
			handler = func(data []byte) {
				m.outputHandler.HandleData(data)
			}
		}
		if err := m.ptyManager.CopyIO(os.Stdin, os.Stdout, os.Stderr, handler, m.inputHandler); err != nil {
			fmt.Fprintf(os.Stderr, "gemini-cli-ntfy: I/O error: %v\n", err)
		}
	}()

	// Setup signal forwarding
	m.setupSignalForwarding()

	return nil
}

// Wait waits for the process to exit
func (m *Manager) Wait() error {
	if m.ptyManager == nil {
		return fmt.Errorf("process not started")
	}

	err := m.ptyManager.Wait()

	m.mu.Lock()
	if m.ptyManager.ProcessState() != nil {
		m.exitCode = m.ptyManager.ProcessState().ExitCode()
	}
	m.mu.Unlock()

	// Ensure terminal is restored
	_ = m.ptyManager.Stop()

	// Signal that we're done
	close(m.done)

	// Cleanup signal handling
	m.cleanupSignals()

	return err
}

// ExitCode returns the exit code of the process
func (m *Manager) ExitCode() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.exitCode
}

// setupSignalForwarding sets up signal forwarding to the child process
func (m *Manager) setupSignalForwarding() {
	m.sigChan = make(chan os.Signal, 1)
	signal.Notify(m.sigChan,
		syscall.SIGTERM,
		syscall.SIGINT,
		syscall.SIGHUP,
		syscall.SIGQUIT,
		syscall.SIGUSR1,
		syscall.SIGUSR2,
		syscall.SIGWINCH,
	)

	go m.forwardSignals()
}

// forwardSignals forwards signals to the child process
func (m *Manager) forwardSignals() {
	for {
		select {
		case sig := <-m.sigChan:
			if m.ptyManager != nil && m.ptyManager.Process() != nil {
				// Forward the signal to the child process
				if err := m.ptyManager.Process().Signal(sig); err != nil {
					// Process might have already exited, but log it
					if err != os.ErrProcessDone {
						fmt.Fprintf(os.Stderr, "gemini-cli-ntfy: signal forward error: %v\n", err)
					}
				}
			}
		case <-m.done:
			return
		}
	}
}

// cleanupSignals stops signal forwarding
func (m *Manager) cleanupSignals() {
	if m.sigChan != nil {
		signal.Stop(m.sigChan)
		close(m.sigChan)
	}
}

// Stop gracefully stops the manager and cleans up resources
func (m *Manager) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ptyManager != nil {
		// Ensure terminal is restored
		_ = m.ptyManager.Stop()

		if m.ptyManager.Process() != nil {
			// Send SIGTERM first for graceful shutdown
			if err := m.ptyManager.Process().Signal(syscall.SIGTERM); err != nil {
				// If SIGTERM fails, force kill
				if err != os.ErrProcessDone {
					return m.ptyManager.Process().Kill()
				}
			}
		}
	}

	return nil
}