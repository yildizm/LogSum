# Contributing to LogSum

Thank you for your interest in contributing to LogSum! This document provides guidelines for development and contribution.

## Development Setup

### Prerequisites
- **Go 1.24+**
- **Git**
- **golangci-lint** (optional, for linting)

### Getting Started

1. **Fork and Clone**
   
   **On Unix/macOS/Linux:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/LogSum.git
   cd LogSum
   ```
   
   **On Windows:**
   ```cmd
   git clone https://github.com/YOUR_USERNAME/LogSum.git
   cd LogSum
   ```

2. **Install Your Development Version**
   
   **All platforms:**
   ```bash
   go install ./cmd/logsum
   ```
   
   **If Windows has issues with .exe extension:**
   ```cmd
   go build -o logsum.exe .\cmd\logsum
   copy logsum.exe %GOPATH%\bin\
   ```

3. **Verify Installation**
   
   **On Unix/macOS/Linux:**
   ```bash
   logsum version
   # Should show: LogSum development (local-build) built on local-build
   ```
   
   **On Windows:**
   ```cmd
   logsum version
   REM Should show: LogSum development (local-build) built on local-build
   
   REM If emojis show as question marks, use:
   logsum --no-emoji version
   
   REM Or improve Windows emoji support:
   REM 1. Install Windows Terminal from Microsoft Store
   REM 2. Install Cascadia Code font: winget install Microsoft.CascadiaCode
   REM 3. Set UTF-8 encoding: chcp 65001
   ```

### PATH Setup (If Command Not Found)

If `logsum` command is not found after installation, add Go's bin directory to your PATH:

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

**Windows (Command Prompt):**
```cmd
REM Get your Go path first
go env GOPATH

REM Add to PATH permanently (default Go path is %USERPROFILE%\go)
setx PATH "%PATH%;%USERPROFILE%\go\bin"
```

**Windows (PowerShell):**
```powershell
# Add to PATH permanently (default Go path is $env:USERPROFILE\go)
$env:PATH += ";$env:USERPROFILE\go\bin"
[Environment]::SetEnvironmentVariable("PATH", $env:PATH, [EnvironmentVariableTarget]::User)
```

## Development Workflow

### Making Changes

1. **Create a Feature Branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make Your Changes**
   ```bash
   # Edit source code
   vim internal/analyzer/matcher.go
   ```

3. **Test Your Changes**
   ```bash
   # Run tests
   go test ./...
   
   # Run tests with coverage
   go test -v -race -coverprofile=coverage.out ./...
   go tool cover -func=coverage.out
   ```

4. **Lint Your Code** (if you have golangci-lint)
   ```bash
   golangci-lint run
   ```

5. **Format Your Code**
   ```bash
   go fmt ./...
   ```

6. **Install and Test Locally**
   ```bash
   go install ./cmd/logsum
   logsum analyze testdata/sample.log
   
   # On Windows, if emojis display incorrectly:
   logsum --no-emoji analyze testdata/sample.log
   ```

### Testing

#### Running Tests
```bash
# All tests
go test ./...

# Specific package
go test ./internal/analyzer/

# With verbose output
go test -v ./...

# With race detection
go test -race ./...
```

#### Test Data
Use the provided test data in `testdata/` for manual testing:
```bash
logsum analyze testdata/sample.log
logsum analyze --no-tui testdata/sample.log

# On Windows, test emoji compatibility:
logsum --no-emoji analyze testdata/sample.log
```

### Code Quality

#### Linting
If you have golangci-lint installed:
```bash
golangci-lint run
```

If you don't have it, you can install it:
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

#### Formatting
Always format your code before committing:
```bash
go fmt ./...
```

## Project Structure

```
LogSum/
├── cmd/logsum/          # Main application entry point
├── internal/
│   ├── analyzer/        # Log analysis engine
│   ├── cli/            # Command-line interface
│   ├── parser/         # Log format parsers
│   └── ui/             # Terminal UI components
├── testdata/           # Test log files
├── configs/            # Default configuration
└── examples/           # Usage examples
```

## Submitting Changes

### Before Submitting

1. **Ensure Tests Pass**
   ```bash
   go test ./...
   ```

2. **Check Code Quality**
   ```bash
   golangci-lint run  # if available
   go fmt ./...
   ```

3. **Test Manually**
   ```bash
   go install ./cmd/logsum
   logsum analyze testdata/sample.log
   
   # Test Windows compatibility if needed:
   logsum --no-emoji analyze testdata/sample.log
   ```

### Pull Request Process

1. **Push Your Branch**
   ```bash
   git push origin feature/your-feature-name
   ```

2. **Create Pull Request**
   - Provide clear description of changes
   - Reference any related issues
   - Include test results if applicable

3. **Review Process**
   - Address any review feedback
   - Ensure CI passes (if configured)

## Release Process (Maintainers Only)

### Creating a Release

1. **Update Version and Tag**
   ```bash
   git tag v1.x.x
   git push origin v1.x.x
   ```

   **Optional: Build with Version Information**
   
   To embed proper version info in releases:
   ```bash
   # Set version variables
   VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v1.0.0")
   COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "none")
   DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
   
   # Build with LDFLAGS
   go build -ldflags "-X main.version=${VERSION} -X main.commit=${COMMIT} -X main.date=${DATE}" -o logsum ./cmd/logsum
   
   # Test version output
   ./logsum version
   # Should show: LogSum v1.0.0 (abc1234) built on 2024-12-01T12:00:00Z
   ```

   **On Windows (PowerShell):**
   ```powershell
   # Get version info
   $VERSION = git describe --tags --abbrev=0 2>$null
   if (!$VERSION) { $VERSION = "v1.0.0" }
   $COMMIT = git rev-parse --short HEAD 2>$null
   if (!$COMMIT) { $COMMIT = "none" }
   $DATE = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ" -AsUTC
   
   # Build with version info
   go build -ldflags "-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE" -o logsum.exe ./cmd/logsum
   
   # Test
   .\logsum.exe version
   ```

2. **Manual Cross-Platform Builds** (if needed)

   **On Unix/macOS/Linux:**
   ```bash
   mkdir -p dist
   
   # Linux
   GOOS=linux GOARCH=amd64 go build -o dist/logsum-linux-amd64 ./cmd/logsum
   
   # macOS Intel
   GOOS=darwin GOARCH=amd64 go build -o dist/logsum-darwin-amd64 ./cmd/logsum
   
   # macOS Apple Silicon
   GOOS=darwin GOARCH=arm64 go build -o dist/logsum-darwin-arm64 ./cmd/logsum
   
   # Windows
   GOOS=windows GOARCH=amd64 go build -o dist/logsum-windows-amd64.exe ./cmd/logsum
   ```

   **On Windows (Command Prompt):**
   ```cmd
   mkdir dist
   
   REM Linux
   set GOOS=linux
   set GOARCH=amd64
   go build -o dist/logsum-linux-amd64 ./cmd/logsum
   
   REM macOS Intel
   set GOOS=darwin
   set GOARCH=amd64
   go build -o dist/logsum-darwin-amd64 ./cmd/logsum
   
   REM macOS Apple Silicon
   set GOOS=darwin
   set GOARCH=arm64
   go build -o dist/logsum-darwin-arm64 ./cmd/logsum
   
   REM Windows
   set GOOS=windows
   set GOARCH=amd64
   go build -o dist/logsum-windows-amd64.exe ./cmd/logsum
   ```

   **On Windows (PowerShell):**
   ```powershell
   New-Item -ItemType Directory -Force -Path dist
   
   # Linux
   $env:GOOS = "linux"; $env:GOARCH = "amd64"
   go build -o dist/logsum-linux-amd64 ./cmd/logsum
   
   # macOS Intel
   $env:GOOS = "darwin"; $env:GOARCH = "amd64"
   go build -o dist/logsum-darwin-amd64 ./cmd/logsum
   
   # macOS Apple Silicon
   $env:GOOS = "darwin"; $env:GOARCH = "arm64"
   go build -o dist/logsum-darwin-arm64 ./cmd/logsum
   
   # Windows
   $env:GOOS = "windows"; $env:GOARCH = "amd64"
   go build -o dist/logsum-windows-amd64.exe ./cmd/logsum
   ```

3. **Create GitHub Release** (optional)
   ```bash
   gh release create v1.x.x dist/* --notes "Release notes here"
   ```

## Getting Help

- **Questions**: Open a GitHub discussion
- **Bugs**: Open a GitHub issue with reproduction steps
- **Feature Requests**: Open a GitHub issue with detailed description

## Code of Conduct

- Be respectful and inclusive
- Focus on constructive feedback
- Help others learn and grow

Thank you for contributing to LogSum! 🚀