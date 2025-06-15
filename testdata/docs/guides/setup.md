---
title: "Setup Guide"
author: "LogSum Documentation Team"
date: "2024-01-10"
tags: ["setup", "installation", "configuration"]
language: "en"
---

# LogSum Setup Guide

This guide walks you through setting up LogSum for your log analysis needs.

## Prerequisites

Before installing LogSum, ensure you have:

- Go 1.21 or later
- At least 2GB of available RAM
- Read access to log files you want to analyze

## Installation Methods

### Option 1: Binary Release

Download the latest binary from our releases page:

```bash
# Linux/macOS
curl -L https://github.com/yildizm/LogSum/releases/latest/download/logsum-$(uname -s)-$(uname -m) -o logsum
chmod +x logsum
sudo mv logsum /usr/local/bin/
```

### Option 2: Go Install

Install directly from source:

```bash
go install github.com/yildizm/LogSum@latest
```

### Option 3: Docker

Run LogSum in a container:

```bash
docker run --rm -v $(pwd):/data ghcr.io/yildizm/logsum:latest analyze /data/app.log
```

## Configuration

### Default Configuration

LogSum works out of the box with sensible defaults. For basic usage:

```bash
logsum analyze /var/log/app.log
```

### Advanced Configuration

Create a configuration file at `~/.logsum/config.yaml`:

```yaml
version: "1.0"

# Pattern detection settings
patterns:
  directories: ["./patterns", "./custom-patterns"]
  auto_reload: true
  enable_defaults: true

# AI analysis settings
ai:
  provider: "ollama"
  model: "llama3.2"
  endpoint: "http://localhost:11434"
  timeout: "30s"

# Storage settings
storage:
  cache_dir: "~/.cache/logsum"
  index_path: "~/.cache/logsum/index.db"
  temp_dir: "/tmp/logsum"

# Output settings
output:
  default_format: "text"
  color_mode: "auto"
  show_progress: true
  verbose: false

# Analysis settings
analysis:
  max_entries: 100000
  timeline_buckets: 60
  enable_insights: true
  timeout: "60s"
```

### Environment Variables

Override settings with environment variables:

```bash
export LOGSUM_AI_PROVIDER=openai
export LOGSUM_AI_API_KEY=your-api-key
export LOGSUM_OUTPUT_FORMAT=json
```

## Pattern Configuration

### Built-in Patterns

LogSum includes patterns for common log formats:
- Apache/Nginx access logs
- Application errors
- Database connection issues
- Performance warnings

### Custom Patterns

Create custom patterns in YAML format:

```yaml
# custom-patterns.yaml
patterns:
  - name: "custom_error"
    regex: "ERROR: (?P<message>.*)"
    severity: "error"
    description: "Custom application errors"
    
  - name: "slow_query"
    regex: "Query took (?P<duration>\\d+)ms"
    severity: "warning"
    threshold: 1000
    description: "Slow database queries"
```

Load custom patterns:

```bash
logsum analyze --patterns ./custom-patterns.yaml /var/log/app.log
```

## Performance Tuning

### Memory Usage

For large log files, adjust memory settings:

```yaml
analysis:
  buffer_size: 8192
  max_line_length: 2097152  # 2MB
```

### Processing Speed

Improve processing speed:

```bash
# Use multiple workers
logsum analyze --workers 8 large-log.log

# Limit analysis scope
logsum analyze --max-entries 50000 large-log.log

# Skip insights for faster processing
logsum analyze --no-insights large-log.log
```

## Troubleshooting

### Common Issues

**Issue**: "Permission denied" errors
**Solution**: Ensure LogSum has read access to log files:
```bash
sudo chmod +r /var/log/app.log
# Or run with appropriate permissions
sudo logsum analyze /var/log/app.log
```

**Issue**: High memory usage
**Solution**: Reduce buffer size or max entries:
```bash
logsum analyze --max-entries 10000 large-file.log
```

**Issue**: Slow analysis
**Solution**: Use streaming mode for large files:
```bash
logsum analyze --stream large-file.log
```

### Debug Mode

Enable debug logging:

```bash
logsum --verbose analyze /var/log/app.log
```

### Getting Help

If you encounter issues:

1. Check the [FAQ](../faq.md)
2. Review [troubleshooting guide](../troubleshooting.md)
3. Open an issue on [GitHub](https://github.com/yildizm/LogSum/issues)

## Next Steps

After setup:

1. Try the [Quick Start tutorial](quickstart.md)
2. Learn about [pattern matching](patterns.md)
3. Explore [output formats](output-formats.md)
4. Set up [real-time monitoring](monitoring.md)