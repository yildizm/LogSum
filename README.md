# LogSum - High-Performance Log Analysis Tool 🚀

[![Go Reference](https://pkg.go.dev/badge/github.com/yildizm/LogSum.svg)](https://pkg.go.dev/github.com/yildizm/LogSum)
[![CI](https://github.com/yildizm/LogSum/workflows/CI/badge.svg)](https://github.com/yildizm/LogSum/actions)
[![Go Report Card](https://goreportcard.com/badge/github.com/yildizm/LogSum)](https://goreportcard.com/report/github.com/yildizm/LogSum)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/github/go-mod/go-version/yildizm/LogSum)](https://github.com/yildizm/LogSum)
[![Release](https://img.shields.io/github/v/release/yildizm/LogSum)](https://github.com/yildizm/LogSum/releases)
[![Issues](https://img.shields.io/github/issues/yildizm/LogSum)](https://github.com/yildizm/LogSum/issues)
[![Stars](https://img.shields.io/github/stars/yildizm/LogSum)](https://github.com/yildizm/LogSum/stargazers)
[![Maintenance](https://img.shields.io/badge/Maintained%3F-yes-green.svg)](https://github.com/yildizm/LogSum/graphs/commit-activity)
[![Performance](https://img.shields.io/badge/Performance-High-brightgreen.svg)](https://github.com/yildizm/LogSum#performance)

Fast log analysis with terminal UI.

LogSum finds patterns and errors in your logs. It supports JSON, logfmt, and plain text formats.

Check out [the blog post]([/guides/content/editing-an-existing-page](https://dev.to/yildizmust4f4/building-logsum-a-33ms-log-analyzer-with-a-beautiful-terminal-ui-13ea)) for the details.

## Features

- Pattern detection with regex
- A pretty Terminal UI
- JSON, logfmt, and text support  
- Real-time file watching
- Timeline view
- Multiple output formats

![Demo - Made with VHS](https://vhs.charm.sh/vhs-5qzxrHcyp8zyeK8Zx5qVEh.gif)

## 🚀 Quick Start

### Installation

#### Method 1: Install Latest Release (Recommended)

```bash
go install github.com/yildizm/LogSum/cmd/logsum@latest
```

#### Method 2: Install Specific Version

```bash
go install github.com/yildizm/LogSum/cmd/logsum@v0.1.0
```

#### Method 3: Build from Source

```bash
git clone https://github.com/yildizm/LogSum.git
cd LogSum
go install ./cmd/logsum
```

> **Windows Terminal Compatibility:** If emojis display as question marks, you have several options:
> 
> **Option 1: Use the --no-emoji flag (Recommended)**
> ```cmd
> logsum --no-emoji analyze app.log
> ```
>
> **Option 2: Improve emoji support**
> - Use **Windows Terminal** (install from Microsoft Store) instead of Command Prompt
> - Install a font with emoji support like **Cascadia Code**:
>   ```cmd
>   # Install via winget (Windows Package Manager)
>   winget install Microsoft.CascadiaCode
>   ```
> - Set terminal to use UTF-8 encoding:
>   ```cmd
>   chcp 65001
>   ```

### PATH Setup

After installation, make sure `$(go env GOPATH)/bin` is in your PATH:

#### Test Installation
```bash
# Check if logsum is available
logsum version
```

#### If Command Not Found, Add Go's bin Directory to PATH:

**macOS/Linux (bash):**
```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
source ~/.bashrc
```

**macOS/Linux (zsh):**
```bash
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.zshrc
source ~/.zshrc
```

**macOS/Linux (fish):**
```bash
fish_add_path (go env GOPATH)/bin
```

**Windows (Command Prompt):**
```cmd
REM Get your Go path first
go env GOPATH

REM Add to PATH permanently (default Go path is %USERPROFILE%\go)
setx PATH "%PATH%;%USERPROFILE%\go\bin"

REM For current session only
set PATH=%PATH%;%USERPROFILE%\go\bin
```

**Windows (PowerShell):**
```powershell
# Get your Go path first
go env GOPATH

# Add to PATH permanently (default Go path is $env:USERPROFILE\go)
$env:PATH += ";$env:USERPROFILE\go\bin"
[Environment]::SetEnvironmentVariable("PATH", $env:PATH, [EnvironmentVariableTarget]::User)

# For current session only
$env:PATH += ";$env:USERPROFILE\go\bin"
```

**After updating PATH, restart your terminal and test:**
```bash
logsum version
```

### Requirements

- **Go 1.24+** (for installation and building from source)

### Basic Usage

**Unix/macOS/Linux:**
```bash
# Analyze a log file
logsum analyze /var/log/app.log

# Beautiful TUI mode (default)
logsum analyze /var/log/app.log

# Real-time monitoring
logsum watch /var/log/app.log

# Pipe logs from stdin
tail -f /var/log/app.log | logsum analyze -

# JSON output for automation
logsum analyze --output json /var/log/app.log > report.json
```

**Windows:**
```cmd
REM Analyze a log file (note: use forward slashes or escape backslashes)
logsum analyze .\testdata\sample.log
logsum analyze testdata/sample.log

REM Real-time monitoring
logsum watch .\app.log

REM JSON output for automation
logsum analyze --output json .\app.log > report.json

REM Common mistake - don't add trailing backslash:
REM WRONG: logsum analyze .\testdata\sample.log\
REM RIGHT: logsum analyze .\testdata\sample.log
```

## 📖 Usage Examples

### Analyze Application Logs

```bash
# Basic analysis with beautiful TUI (default)
logsum analyze /var/log/myapp.log

# Text output mode
logsum analyze --no-tui /var/log/myapp.log

# JSON output for automation
logsum analyze --no-tui --output json /var/log/myapp.log
```

### Real-time Monitoring

```bash
# Watch logs in real-time
logsum watch /var/log/nginx/access.log

# Watch with custom patterns
logsum watch --patterns ./patterns/ /var/log/app.log
```

### Output Formats

```bash
# Human-readable text (default)
logsum analyze /var/log/app.log

# JSON for automation
logsum analyze --no-tui --output json /var/log/app.log

# Markdown for documentation
logsum analyze --no-tui --output markdown /var/log/app.log > report.md
```

### Custom Patterns

```bash
# Use custom pattern files
logsum analyze --patterns /path/to/patterns.yaml /var/log/app.log

# Use custom pattern directory
logsum analyze --patterns /path/to/pattern-files/ /var/log/app.log

# Copy and customize default patterns
cp examples/patterns/default.yaml my-patterns.yaml
# Edit my-patterns.yaml to add your custom patterns
logsum analyze --patterns my-patterns.yaml /var/log/app.log
```

### Advanced Features

```bash
# Disable emojis for Windows compatibility
logsum --no-emoji analyze /var/log/app.log

# Check available flags
logsum analyze --help
```

## 🔧 Configuration

LogSum supports YAML configuration files for custom pattern matching:

```yaml
# patterns.yaml
patterns:
  error:
    - regex: "ERROR|error|Error"
      severity: high
      description: "Error messages"
  
  performance:
    - regex: "took \\d+ms|duration: \\d+"
      severity: medium
      description: "Performance metrics"
  
  security:
    - regex: "failed login|unauthorized|forbidden"
      severity: critical
      description: "Security events"

formats:
  - type: json
    timestamp_field: "@timestamp"
    level_field: "level"
  
  - type: logfmt
    timestamp_format: "2006-01-02T15:04:05Z"
```

Use with: `logsum analyze --config patterns.yaml /var/log/app.log`

## How it works

LogSum scans your logs for common patterns like errors, timeouts, and performance issues. It groups entries by time to show you when problems happened.

## 🎨 TUI Interface

The interactive TUI provides:

- **Timeline View** - Visual representation of log patterns over time
- **Pattern Statistics** - Frequency and distribution of detected patterns
- **Log Viewer** - Searchable, filterable log entries with syntax highlighting
- **Insights Panel** - Smart suggestions and automated issue identification
- **Real-time Updates** - Live monitoring with automatic refresh

### Keyboard Shortcuts

- `q` - Quit
- `↑/↓` - Navigate lists
- `/` - Search
- `f` - Filter
- `r` - Refresh
- `Tab` - Switch panels
- `Enter` - Select/View details

## Development

Want to contribute? See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup and guidelines.

### Quick Development Setup
```bash
git clone https://github.com/yildizm/LogSum.git
cd LogSum
go install ./cmd/logsum

# Test your changes
go test ./...
logsum analyze testdata/sample.log
```

## Performance

LogSum is fast. It can process large log files quickly and doesn't use much memory.

Benchmarks on MacBook Pro M2:
```
10K entries:     3.3ms
100K entries:    28ms
1M entries:      245ms
10M entries:     2.1s
```
## 📜 License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI
- Uses [Cobra](https://github.com/spf13/cobra) for CLI
- Terminal formatting powered by [go-termfmt](https://github.com/yildizm/go-termfmt) - Beautiful terminal formatting library
- Log parsing powered by [go-logparser](https://github.com/yildizm/go-logparser) - High-performance log parsing library
