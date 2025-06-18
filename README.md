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

**From Solo Developer to Enterprise Teams: LogSum evolves with your needs**

LogSum is a high-performance log analysis tool that scales from individual development to enterprise incident response. It bridges the gap between logs, team documentation, and AI-powered insights.

Whether you're debugging locally, correlating errors with team docs, or using AI for intelligent analysis, LogSum adapts to your workflow.

Check out [the blog post](https://dev.to/yildizmust4f4/building-logsum-a-33ms-log-analyzer-with-a-beautiful-terminal-ui-13ea) for the details.

## Features

- **Pattern Detection** - Advanced regex-based pattern matching with machine learning insights
- **AI-Powered Analysis** - GPT/Claude/Ollama integration for intelligent error analysis and recommendations
- **RAG Pipeline** - Retrieval-Augmented Generation with document correlation for context-aware insights
- **Vector Search** - Semantic search through documentation and knowledge bases (6.7x performance improvement)
- **Beautiful Terminal UI** - Interactive TUI with timeline visualization and real-time updates
- **Multiple Formats** - JSON, logfmt, and plain text support with auto-detection
- **⏱Real-time Monitoring** - File watching with live analysis and alerting
- **Performance Monitoring** - Built-in metrics collection and reporting system
- **Performance** - Sub-100ms analysis with optimized caching and concurrent processing
- **Flexible Output** - JSON, Markdown, CSV, and text formats for automation and reporting
- **Enterprise Ready** - Modular architecture, comprehensive testing, and Go best practices

![Demo - Made with VHS](https://vhs.charm.sh/vhs-5qzxrHcyp8zyeK8Zx5qVEh.gif)

## 🚀 Quick Start

### Installation

#### Method 1: Install Latest Release (Recommended)

```bash
go install github.com/yildizm/LogSum/cmd/logsum@latest
```

#### Method 2: Install Specific Version

```bash
go install github.com/yildizm/LogSum/cmd/logsum@v0.3.0
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

## 🎯 Three Ways to Use LogSum

LogSum evolved to serve different needs as teams and projects grow:

### 1. **Solo Developer: Fast Local Analysis**
*Perfect for debugging your own code*

```bash
# Quick analysis with beautiful terminal UI
logsum analyze /var/log/app.log

# Export structured reports
logsum analyze --output json /var/log/app.log > report.json
```

**Use this when:**
- Debugging local development issues
- Analyzing application logs during development  
- Need fast pattern detection and insights
- Want beautiful terminal visualization

---

### 2. **Team Collaboration: Documentation Correlation**
*Bridge logs with team knowledge*

```bash
# Correlate errors with your team's documentation
logsum analyze --correlate --docs ./team-docs/ /var/log/app.log
```

**Use this when:**
- Working in cross-functional teams
- Have troubleshooting docs, API documentation, runbooks
- Want to automatically link errors to relevant team knowledge
- Need faster incident response with contextual information

**Example:** Error `"Database connection timeout"` automatically shows related docs:
- `docs/database/troubleshooting.md` 
- `docs/deployment/connection-pools.md`
- `docs/runbooks/database-issues.md`

---

### 3. **Enterprise: AI-Powered Analysis**  
*Intelligent insights with team context*

#### Setup AI Providers

**Option A: Ollama (Recommended - Free & Local)**
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh

# Pull a model (choose one)
ollama pull llama3.2          # Fast, good quality
ollama pull llama3.2:13b      # Better quality, more resources
ollama pull codellama         # Code-focused model

# Verify installation
ollama list
```

**Option B: OpenAI**
```bash
# Get API key from https://platform.openai.com/api-keys
export OPENAI_API_KEY="your-api-key-here"
```

#### Configure LogSum
```bash
# Generate config file
logsum config init

# Edit .logsum.yaml:
# ai:
#   provider: "ollama"     # or "openai"
#   model: "llama3.2"      # or "gpt-4"
#   endpoint: "http://localhost:11434"  # for ollama
#   api_key: ""            # for openai (or use env var)

# Get AI analysis with document context
logsum analyze --ai --correlate --docs ./docs/ /var/log/app.log

# Monitor performance during analysis
logsum monitor start --duration 60s &
logsum analyze --ai /var/log/app.log
logsum monitor report --format json > metrics.json
```

**Use this when:**
- Managing complex distributed systems
- Need intelligent error analysis and recommendations
- Want AI to understand your team's specific context and documentation
- Have multiple services with extensive documentation

**Example:** AI analyzes `"Redis connection failed"` with context from your team's Redis documentation and suggests specific troubleshooting steps from your runbooks.

#### Troubleshooting AI Setup

**Ollama Connection Issues:**
```bash
# Check if Ollama is running
curl http://localhost:11434/api/tags

# Start Ollama if not running
ollama serve

# Test with LogSum
logsum analyze --ai --no-tui /var/log/app.log
```

**Common Errors:**
- `connection refused`: Ollama isn't running → `ollama serve`
- `model not found`: Pull the model → `ollama pull llama3.2`  
- `API key invalid`: Check OpenAI key → verify at platform.openai.com

---

## 📖 Detailed Examples

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

### Document Correlation (v0.3.0+)

```bash
# Enable document correlation with documentation
logsum analyze --correlate --docs ./knowledge-base/ /var/log/app.log

# AI-enhanced correlation analysis
logsum analyze --ai --correlate --docs ./docs/ /var/log/app.log

# Output to file
logsum analyze --correlate --docs ./docs/ --output-file report.md /var/log/app.log

# Use with custom patterns
logsum analyze --patterns ./patterns/ --correlate --docs ./docs/ /var/log/app.log
```

### Configuration Management

```bash
# Generate sample configuration
logsum config init

# Show current configuration
logsum config show

# Validate configuration
logsum config validate

# Show config file paths
logsum config path
```

### Available Commands

```bash
# Core analysis
logsum analyze /var/log/app.log              # Basic analysis
logsum analyze --ai /var/log/app.log          # AI-powered analysis
logsum analyze --correlate --docs ./docs/ /var/log/app.log  # Document correlation

# Pattern management
logsum patterns list           # List available patterns
logsum patterns validate       # Validate pattern files

# Real-time monitoring
logsum watch /var/log/app.log  # Watch file for changes

# Performance monitoring (NEW)
logsum monitor start --duration 30s          # Start metrics collection
logsum monitor report --format json          # Generate performance report
logsum monitor stop                           # Stop monitoring

# Configuration
logsum config init             # Create sample config
logsum config show             # Display current config
logsum config validate         # Validate config file
logsum config path             # Show config file locations

# Utility options
logsum --no-emoji analyze /var/log/app.log   # Disable emojis for Windows
logsum --version                              # Show version info
logsum analyze --help                         # Show command help
```

## 🔧 Configuration

LogSum uses YAML configuration for all advanced features:

```bash
# Generate complete configuration template
logsum config init

# This creates .logsum.yaml with all options
```

### Key Configuration Sections

```yaml
# Basic analysis settings
analysis:
  timeout: 30s
  max_entries: 100000

# Document correlation (for teams)
correlation:
  enabled: true
  docs_path: "./docs"
  
# AI configuration (for enterprise)
ai:
  provider: "ollama"     # ollama, openai
  model: "llama3.2"      # or gpt-4, gpt-3.5-turbo
  endpoint: "http://localhost:11434"
  api_key: ""            # required for OpenAI
  timeout: 30s
  
# Performance monitoring
monitor:
  enabled: true
  interval: 5s
  include_system: true
  include_operations: true
  
# Custom patterns
patterns:
  directories: ["./patterns"]
  enable_defaults: true
```

Use with: `logsum analyze --config .logsum.yaml /var/log/app.log`

### Full Configuration Example

```yaml
# logsum.yaml - Complete configuration
correlation:
  enabled: true
  docs_path: "./docs"
  max_correlations: 5
  similarity_threshold: 0.7

analysis:
  timeout: 30s
  max_entries: 100000
  enable_timeline: true

output:
  format: "text"  # text, json, markdown, csv
  verbose: false
  color_mode: "auto"  # auto, always, never
  
patterns:
  directories:
    - "./patterns"
    - "~/.logsum/patterns"
  enable_defaults: true
  
  custom_patterns:
    database_errors:
      pattern: "database.*error|connection.*failed"
      severity: "error"
      description: "Database connectivity issues"
```

Use with: `logsum analyze --config logsum.yaml /var/log/app.log`

## How it works

LogSum scans your logs for common patterns like errors, timeouts, and performance issues. It groups entries by time to show you when problems happened.

## TUI Interface

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

LogSum v0.3.0 delivers exceptional performance with sub-100ms analysis for typical workloads and optimized memory usage. The latest improvements include:

- **6.7x Vector Caching Performance** - Optimized vector storage with intelligent caching
- **Modular Architecture** - Clean separation of concerns for better maintainability
- **Enhanced Error Handling** - Comprehensive error management and recovery
- **Performance Monitoring** - Built-in metrics collection and reporting
- **Code Quality** - Follows Go best practices with comprehensive testing

### Core Analysis Benchmarks (MacBook Pro M1 Pro)
```
Operation                    Time        Memory      Notes
─────────────────────────────────────────────────────────
10K entries analysis         3.3ms       8MB         Pattern detection
100K entries analysis        28ms        45MB        Full timeline
1M entries analysis          245ms       180MB       Large dataset
Vector search (1K docs)      570μs       74KB        Similarity search
AI analysis (small)          96μs        -           Mock provider
Cache hit ratio              95%         -           Repeated queries
```

### Vector Store Performance (v0.3.0)
```
Operation                    Time        Memory      Scalability
─────────────────────────────────────────────────────────
Store vector (384-dim)       197ns       76B         Linear
Search 1000 vectors          570μs       74KB        O(n)
Cached similarity            442ns       0B          Constant
Vector normalization         726ns       1.5KB       Linear
TF-IDF vectorization         ~2ms        ~100KB      Corpus-dependent
```

### RAG Pipeline Benchmarks
```
Component                    Time        Memory      Notes
─────────────────────────────────────────────────────────
Document indexing            <10ms       <1MB        Per 100 docs
Correlation analysis         <50ms       <5MB        Pattern matching
AI context building          <20ms       <2MB        Token optimization
End-to-end pipeline          <100ms      <10MB       Complete analysis
```

**Performance Targets Met:**
- Typical analysis: <100ms (achieved: 96μs - 570μs)
- Memory usage: <1MB for 1K vectors (achieved: 74KB)  
- Vector caching: 6.7x performance improvement
- Concurrent operations: Thread-safe with RWMutex
- Cache effectiveness: 95%+ hit ratio for repeated queries
- Code quality: 100% linting compliance, comprehensive test coverage

See [PERFORMANCE.md](PERFORMANCE.md) for detailed benchmarks and optimization strategies.
## 📜 License

MIT License - see [LICENSE](LICENSE) for details.

## Acknowledgments

- Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) for the TUI
- Uses [Cobra](https://github.com/spf13/cobra) for CLI
- Terminal formatting powered by [go-termfmt](https://github.com/yildizm/go-termfmt) - Beautiful terminal formatting library
- Log parsing powered by [go-logparser](https://github.com/yildizm/go-logparser) - High-performance log parsing library
