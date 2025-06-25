package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nakkulla/gemini-cli-ntfy/pkg/config"
	"github.com/nakkulla/gemini-cli-ntfy/pkg/interfaces"
	"github.com/nakkulla/gemini-cli-ntfy/pkg/monitor"
	"github.com/nakkulla/gemini-cli-ntfy/pkg/notification"
	"github.com/nakkulla/gemini-cli-ntfy/pkg/process"
)

// Dependencies holds all the dependencies for the application
type Dependencies struct {
	Config         *config.Config
	Notifier       notification.Notifier
	OutputMonitor  interfaces.DataHandler
	ProcessManager *process.Manager
	stopChan       chan struct{}
}

// NewDependencies creates all dependencies with the given configuration
func NewDependencies(cfg *config.Config) (*Dependencies, error) {
	deps := &Dependencies{
		Config:   cfg,
		stopChan: make(chan struct{}),
	}

	// Create notification components
	baseNotifier := notification.NewNtfyClient(cfg.NtfyServer, cfg.NtfyTopic)

	// Create output monitor with stdout notifier temporarily
	outputMonitor := monitor.NewOutputMonitor(cfg, notification.NewStdoutNotifier())

	// Wrap with context notifier
	contextNotifier := notification.NewContextNotifier(baseNotifier, func() string {
		return outputMonitor.GetTerminalTitle()
	})

	// Wrap with backstop notifier if configured
	var finalNotifier notification.Notifier = contextNotifier
	if cfg.BackstopTimeout > 0 {
		finalNotifier = notification.NewBackstopNotifier(contextNotifier, cfg.BackstopTimeout)
	}
	deps.Notifier = finalNotifier

	// Update the output monitor with the final notifier
	outputMonitor.SetNotifier(deps.Notifier)
	deps.OutputMonitor = outputMonitor

	// Create input handler that disables backstop timer
	inputHandler := func() {
		if backstopNotifier, ok := deps.Notifier.(*notification.BackstopNotifier); ok {
			backstopNotifier.DisableBackstopTimer()
			if os.Getenv("GEMINI_NOTIFY_DEBUG") == "true" {
				fmt.Fprintf(os.Stderr, "gemini-cli-ntfy: user input detected, disabling backstop timer\n")
			}
		}
	}

	// Create process manager
	deps.ProcessManager = process.NewManager(cfg, deps.OutputMonitor, inputHandler)

	return deps, nil
}

// Close cleans up all dependencies
func (d *Dependencies) Close() {
	// Stop status indicator refresh
	if d.stopChan != nil {
		select {
		case <-d.stopChan:
			// Already closed
		default:
			close(d.stopChan)
		}
		d.stopChan = nil
	}

	// Close notifiers
	// First try to close as backstop notifier
	if backstopNotifier, ok := d.Notifier.(*notification.BackstopNotifier); ok {
		_ = backstopNotifier.Close()
	}
}

// Application represents the main application
type Application struct {
	deps *Dependencies
}

// NewApplication creates a new application with the given dependencies
func NewApplication(deps *Dependencies) *Application {
	return &Application{
		deps: deps,
	}
}

// Run starts the application with the given command and arguments
func (a *Application) Run(command string, args []string) error {
	// Send startup notification if configured
	if a.deps.Config.StartupNotify && !a.deps.Config.Quiet {
		pwd, _ := os.Getwd()
		startupNotification := notification.Notification{
			Title:   "Gemini CLI Session Started",
			Message: fmt.Sprintf("Working directory: %s", pwd),
			Time:    time.Now(),
			Pattern: "startup",
		}
		_ = a.deps.Notifier.Send(startupNotification)
	}

	if err := a.deps.ProcessManager.Start(command, args); err != nil {
		return err
	}

	return a.deps.ProcessManager.Wait()
}

// Stop gracefully stops the application
func (a *Application) Stop() error {
	return a.deps.ProcessManager.Stop()
}

// ExitCode returns the exit code of the wrapped process
func (a *Application) ExitCode() int {
	return a.deps.ProcessManager.ExitCode()
}