# Test Verbose Logging

## How to Enable Comprehensive Component Logging

### 1. Run with Verbose Flag
```bash
# Now when you run this command with --verbose, you'll see detailed component logging:
logsum analyze /tmp/logsum-errors.log --config logsum-demo.yaml --ai --correlate --docs /tmp/s3-docs --output-file demo-analysis.md -o markdown --no-tui --verbose
```

### 2. Expected Verbose Output
With the new logging system, you'll see detailed output like:

```
[15:23:41.234] INFO [ai-setup] starting correlation analysis [patterns=5 raw_entries=1247]
[15:23:41.235] DEBUG [correlator] initialized correlation result [patterns=5 raw_entries=1247]
[15:23:41.236] DEBUG [correlator] processing pattern-based correlations (5 patterns)
[15:23:41.237] DEBUG [correlator] correlating pattern 1: DatabaseConnectionError
[15:23:41.245] DEBUG [correlator] pattern correlation successful [pattern=DatabaseConnectionError matches=3]
[15:23:41.246] DEBUG [correlator] correlating pattern 2: AuthenticationError
[15:23:41.248] DEBUG [correlator] no documents matched pattern: AuthenticationError
[15:23:41.249] INFO [correlator] pattern correlation completed [total_patterns=5 correlated_patterns=3]
[15:23:41.250] DEBUG [correlator] extracting error entries from raw logs
[15:23:41.267] DEBUG [correlator] error extraction completed [raw_entries=1247 error_entries=89]
[15:23:41.268] INFO [correlator] starting direct error correlation (89 errors)
[15:23:41.456] INFO [correlator] direct error correlation successful [correlated_errors=12]
[15:23:41.457] INFO [correlator] correlation analysis completed [duration=223ms total_patterns=5 correlated_patterns=3 total_errors=89 correlated_errors=12]
```

### 3. Log Format
Each log entry shows:
- **Timestamp**: `[15:23:41.234]`
- **Level**: `INFO`, `DEBUG`, `WARN`, `ERROR`
- **Component**: `[correlator]`, `[ai-setup]`, `[analyzer]`, etc.
- **Message**: Human-readable description
- **Structured Fields**: `[key=value key2=value2]`

### 4. Component Granularity
The logging system provides visibility into:

**Correlation Engine**:
- Pattern matching process
- Document search operations
- Error extraction and classification
- Keyword extraction details
- Vector search operations
- Performance metrics

**AI Analysis**:
- LLM API request/response cycles
- Prompt construction
- JSON parsing results
- Document context building
- Token management

**Vector Store**:
- Embedding calculations
- Similarity searches
- Cache hits/misses
- Index operations

**Document Store**:
- File scanning progress
- Indexing operations
- Search queries
- Memory usage

### 5. Performance Insights
With verbose logging, you can see:
- **Timing**: How long each operation takes
- **Counts**: Number of documents, patterns, errors processed
- **Success/Failure**: Which operations succeed or fail
- **Memory**: Object counts and sizes
- **API Usage**: LLM tokens and requests

### 6. Adding More Logging
To add logging to any component:

```go
// In any component file:
import "github.com/yildizm/LogSum/internal/cli"

// Get component logger
logger := cli.GetLogger("my-component")

// Use throughout the component
logger.Info("operation started")
logger.Debug("processing item %d", i)
logger.DebugWithFields("operation completed", []logger.Field{
    logger.F("items", count),
    logger.Duration(elapsed),
})
```

This provides comprehensive visibility into LogSum's internal operations when using the `--verbose` flag.