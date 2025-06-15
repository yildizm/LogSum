# Performance Optimization

LogSum is designed for high-performance log analysis. This document covers optimization strategies for different use cases.

## Benchmarks

### Standard Performance

On a modern system (16GB RAM, 8 cores):
- **Small files** (< 100MB): ~5 seconds
- **Medium files** (100MB - 1GB): ~30 seconds  
- **Large files** (1GB - 10GB): ~5 minutes
- **Very large files** (> 10GB): Streaming recommended

### Memory Usage

- **Base overhead**: ~50MB
- **Per log entry**: ~500 bytes
- **Index overhead**: ~20% of processed data
- **Peak usage**: 3-4x file size for full analysis

## Optimization Strategies

### For Large Files

Use streaming mode to reduce memory usage:

```bash
logsum analyze --stream --buffer-size 8192 large-file.log
```

Limit the scope of analysis:

```bash
# Analyze only recent entries
logsum analyze --tail 10000 large-file.log

# Skip detailed insights
logsum analyze --no-insights large-file.log
```

### For Real-time Processing

Configure appropriate buffer sizes:

```yaml
analysis:
  buffer_size: 4096
  max_line_length: 65536
```

Use targeted pattern matching:

```bash
logsum watch --patterns error,warning app.log
```

### For Batch Processing

Enable parallel processing:

```bash
# Use all available cores
logsum analyze --workers $(nproc) batch/*.log

# Process multiple files
find /logs -name "*.log" | xargs -P 4 -I {} logsum analyze {}
```

## Configuration Examples

### High-throughput Setup

```yaml
analysis:
  buffer_size: 16384
  max_entries: 1000000
  workers: 8
  
storage:
  cache_dir: "/fast-ssd/logsum-cache"
  
output:
  show_progress: false
```

### Memory-constrained Setup

```yaml
analysis:
  buffer_size: 2048
  max_entries: 50000
  stream_mode: true
  
storage:
  cache_dir: "/tmp/logsum"
```

### Real-time Monitoring

```yaml
analysis:
  buffer_size: 1024
  timeout: "5s"
  
output:
  compact_mode: true
  show_progress: false
```