# gemini-cli-ntfy

A transparent wrapper for Gemini CLI that sends a notification when Gemini needs your attention.

## Features

- 🔔 Single notification when Gemini needs attention
- 🔄 Transparent wrapping - preserves all Gemini CLI functionality
- 💤 Intelligent inactivity detection
- 🖥️ Cross-platform support (Linux/macOS)

### Intelligent Inactivity Detection

The backstop timer provides smart notifications when Gemini might need your attention:

- **While Gemini is outputting**: Timer is continuously reset
- **When Gemini stops**: A 30-second countdown begins
- **If you start typing**: Timer is permanently disabled
- **If Gemini sent a bell**: Timer is disabled (you're already notified)
- **After 30 seconds of inactivity**: ONE notification is sent

This ensures you're notified when Gemini needs input, but not when you're actively working.

## Installation

### Go Install

```bash
go install github.com/nakkulla/gemini-cli-ntfy/cmd/gemini-cli-ntfy@latest
```

### Build from Source

```bash
git clone https://github.com/nakkulla/gemini-cli-ntfy
cd gemini-cli-ntfy
make build
```

## Quick Start

1. Set your ntfy topic:
   ```bash
   export GEMINI_NOTIFY_TOPIC="my-gemini-notifications"
   ```

2. Run Gemini CLI:
   ```bash
   gemini-cli-ntfy
   ```

## Configuration

Configure via environment variables:

- `GEMINI_NOTIFY_TOPIC` - Ntfy topic for notifications (required)
- `GEMINI_NOTIFY_SERVER` - Ntfy server URL (default: https://ntfy.sh)
- `GEMINI_NOTIFY_BACKSTOP_TIMEOUT` - Inactivity timeout (default: 30s)
- `GEMINI_NOTIFY_QUIET` - Disable notifications (true/false)
- `GEMINI_NOTIFY_GEMINI_PATH` - Path to the real gemini binary

Or use a config file at `~/.config/gemini-cli-ntfy/config.yaml`:

```yaml
ntfy_topic: "my-gemini-notifications"
ntfy_server: "https://ntfy.sh"
backstop_timeout: "30s"
quiet: false
gemini_path: "/usr/local/bin/gemini"
```

## Development

Simple development workflow:

```bash
# One-time setup
make install-tools  # Install required development tools

# During development
make test          # Run tests with race detection

# Before committing
make fix           # Auto-fix issues
make verify        # Run all checks
```

## Usage Examples

### Basic Usage
```bash
# Start with notifications enabled
export GEMINI_NOTIFY_TOPIC="my-notifications"
gemini-cli-ntfy

# Or pass arguments directly to Gemini
gemini-cli-ntfy --help
```

### Custom Configuration
```bash
# Use custom timeout
export GEMINI_NOTIFY_BACKSTOP_TIMEOUT="60s"
gemini-cli-ntfy

# Quiet mode (no notifications)
gemini-cli-ntfy --quiet
```

## Architecture

Based on claude-code-ntfy, this wrapper:

1. **Detects Gemini CLI binary** in PATH (excluding self)
2. **Wraps with PTY** for transparent terminal emulation
3. **Monitors output** for activity and bell characters
4. **Tracks user input** to disable unnecessary notifications
5. **Sends notifications** via ntfy.sh after periods of inactivity

## License

MIT

## Credits

Based on [claude-code-ntfy](https://github.com/Veraticus/claude-code-ntfy) by Veraticus.