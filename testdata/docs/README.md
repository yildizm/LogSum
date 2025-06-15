# LogSum Documentation

Welcome to the LogSum documentation repository. This contains comprehensive guides, API documentation, and examples for using LogSum effectively.

## Quick Start

LogSum is a high-performance log analysis tool that automatically detects patterns, identifies anomalies, and provides insights from your log data.

### Installation

```bash
go install github.com/yildizm/LogSum@latest
```

### Basic Usage

```bash
# Analyze a log file
logsum analyze /path/to/logfile.log

# Watch logs in real-time
logsum watch /path/to/logfile.log

# Generate insights
logsum analyze --insights /path/to/logfile.log
```

## Features

- **Pattern Detection**: Automatically identifies log patterns
- **Anomaly Detection**: Finds unusual log entries
- **Real-time Analysis**: Monitor logs as they're written
- **Multiple Formats**: Supports JSON, logfmt, and plain text
- **Performance Insights**: Provides timeline and statistics
- **Configurable Output**: JSON, CSV, Markdown, and terminal formats

## Documentation Structure

- [API Reference](api/) - Complete API documentation
- [User Guides](guides/) - Step-by-step tutorials
- [Examples](../examples/) - Sample configurations and use cases

## Contributing

See our [contributing guidelines](CONTRIBUTING.md) for information on how to contribute to this project.