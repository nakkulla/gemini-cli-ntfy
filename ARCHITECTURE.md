# Gemini CLI Ntfy - Architecture

## Overview

Gemini CLI Ntfy is a transparent wrapper for Gemini CLI that sends a notification when Gemini needs your attention. The wrapper preserves all Gemini CLI functionality while adding intelligent inactivity detection that respects user awareness.

### Goals
- Zero-impact transparent wrapping of Gemini CLI
- Smart inactivity detection with user input awareness  
- Single notification per idle period
- Cross-platform support (Linux/macOS)
- Simple, focused functionality

### Non-Goals
- Pattern matching or regex-based notifications
- Complex notification rules or conditions
- Rate limiting or batching multiple notifications
- Status indicators or UI elements
- Modifying Gemini CLI behavior

## Architecture

### Simplified Architecture

```
┌─────────────────┐
│   User Input    │
└────────┬────────┘
         │
         ▼
┌─────────────────────────────────────────┐
│           gemini-cli-ntfy               │
│                                         │
│  ┌─────────────┐    ┌────────────────┐ │
│  │   Config    │    │    Process     │ │
│  │   Loader    │    │    Manager     │ │
│  └─────────────┘    └───────┬────────┘ │
│                             │          │
│  ┌─────────────┐    ┌───────┴────────┐ │
│  │   Output    │◄───┤      PTY       │ │
│  │   Monitor   │    │    Manager     │ │
│  └──────┬──────┘    └───────┬────────┘ │
│         │                   │          │
│         ▼                   ▼          │
│  ┌─────────────┐    ┌────────────────┐ │
│  │  Backstop   │    │ Input Handler  │ │
│  │  Notifier   │◄───┤   (stdin)      │ │
│  └──────┬──────┘    └────────────────┘ │
│         │                              │
│         ▼                              │
│  ┌─────────────┐                      │
│  │ Ntfy Client │                      │
│  └─────────────┘                      │
└─────────────────────────────────────────┘
         │
         ▼
┌─────────────────┐
│  Gemini CLI     │
└─────────────────┘
```

### Component Responsibilities

1. **Config Loader**: Loads configuration from environment variables and/or YAML config file
2. **Process Manager**: Manages Gemini CLI subprocess lifecycle using PTY
3. **PTY Manager**: Provides transparent terminal emulation with separate stdin/stdout handling
4. **Output Monitor**: Tracks Gemini output activity and detects bell characters
5. **Input Handler**: Detects user stdin activity to disable backstop timer
6. **Backstop Notifier**: Sends ONE notification after 30s of Gemini inactivity
7. **Ntfy Client**: HTTP client that sends notifications to ntfy.sh

## Key Design Decisions

### 1. Inactivity Detection Logic
The backstop timer implements smart notification behavior:

- **While Gemini is outputting**: Timer continuously resets
- **When Gemini stops**: 30-second countdown begins
- **If user types (stdin)**: Timer is permanently disabled until Gemini responds
- **If Gemini sends bell**: Timer is permanently disabled (user already notified)
- **After 30 seconds**: ONE notification sent

Key insight: User input indicates awareness that Gemini is waiting, so no notification needed.

### 2. Simplified Architecture
- **No pattern matching**: Removed all regex functionality
- **Single notification type**: "Gemini needs attention"
- **Binary state**: Gemini is either active or waiting
- **User awareness tracking**: Stdin activity = user knows

### 3. Implementation Approach
- Monitor stdin separately from PTY output using input handler
- Track Gemini output for activity (any output resets timer)
- Disable timer permanently on user interaction
- Reset everything when Gemini responds (new session)

## Package Structure

```
pkg/
├── config/          # Configuration loading and validation
│   └── config.go    # Config struct and loading logic
├── process/         # Process management and PTY handling
│   ├── manager.go   # Process lifecycle management
│   ├── pty.go       # PTY creation and I/O handling
│   └── interfaces.go # PTY interface definition
├── monitor/         # Output monitoring
│   ├── output_monitor.go     # Activity tracking and bell detection
│   ├── terminal_detector.go  # Terminal sequence detection
│   └── terminal_state.go     # Terminal state management
├── notification/    # Notification system
│   ├── notification.go      # Notification type
│   ├── backstop_notifier.go # Inactivity timer logic
│   ├── ntfy_client.go       # HTTP client for ntfy.sh
│   └── stdout_notifier.go   # Testing/debug notifier
├── interfaces/      # Core interface definitions
│   └── interfaces.go        # Shared interfaces
└── testutil/        # Testing utilities
    └── mocks.go             # Mock implementations
```

## Configuration

### Config File Format
```yaml
# ~/.config/gemini-cli-ntfy/config.yaml

# Notification settings
ntfy_topic: "my-gemini-notifications"
ntfy_server: "https://ntfy.sh"

# Backstop timeout (default: 30s)
backstop_timeout: "30s"

# Disable all notifications
quiet: false

# Path to real gemini binary (auto-detected if not set)
gemini_path: "/usr/local/bin/gemini"
```

### Environment Variables
```bash
export GEMINI_NOTIFY_TOPIC="my-topic"
export GEMINI_NOTIFY_SERVER="https://ntfy.sh"
export GEMINI_NOTIFY_BACKSTOP_TIMEOUT="30s"
export GEMINI_NOTIFY_QUIET="false"
export GEMINI_NOTIFY_GEMINI_PATH="/usr/local/bin/gemini"
```

### Configuration Priority
1. Command-line flags (highest priority)
2. Environment variables
3. Config file
4. Default values (lowest priority)

## Implementation Details

### Process Management
- Uses `github.com/creack/pty` for PTY creation
- Full signal forwarding (SIGTERM, SIGINT, SIGWINCH, etc.)
- Transparent I/O copying with separate stdin/stdout handlers
- Terminal size synchronization
- Self-wrap detection via environment variable

### Output Monitoring
- Tracks last output time for activity detection
- Line buffering for bell detection
- Terminal sequence detection for screen clears (new session)
- Thread-safe with mutex protection
- Sends activity signals to backstop notifier

### Backstop Notification
- Single timer per session
- Resets on any Gemini output
- Permanently disabled by user input
- Permanently disabled by bell character
- Only sends ONE notification per idle period
- Resets on screen clear (new prompt)

### PTY I/O Handling
- Separate handlers for input and output
- Input detection via wrapper Reader
- Output monitoring via wrapper Reader
- Preserves raw terminal mode
- Full transparency for all terminal features

## Performance Characteristics

- **Startup time**: < 50ms
- **Memory usage**: ~8MB RSS (without Gemini CLI)
- **CPU usage**: < 0.1% when idle
- **I/O latency**: < 1ms (transparent passthrough)
- **Zero impact on Gemini CLI performance**

## Security Considerations

1. **Command Injection**: Using `exec.Command` with separate args
2. **PTY Security**: Proper cleanup and signal handling
3. **No Logging**: Output is never logged or stored
4. **Ntfy Authentication**: Supports auth tokens if needed
5. **Signal Handling**: Proper cleanup on all termination signals

## Credits

Based on [claude-code-ntfy](https://github.com/Veraticus/claude-code-ntfy) by Veraticus.