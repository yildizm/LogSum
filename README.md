# LogSum - AI-Powered Log Analyzer

**Stop wasting hours digging through logs. Get insights in seconds.**

LogSum analyzes your logs and tells you exactly what went wrong, why it happened, and how to fix it. Powered by AI and your team's documentation.

```bash
# Install and analyze in 30 seconds
go install github.com/yildizm/LogSum/cmd/logsum@latest
logsum analyze /var/log/app.log
```

![Demo](https://vhs.charm.sh/vhs-5qzxrHcyp8zyeK8Zx5qVEh.gif)

## Picture This

It's 3 AM. Your phone buzzes with alerts. Production is down. You're bleary-eyed, ssh'ing into servers, tailing logs, trying to piece together what went wrong from thousands of cryptic error messages.

**What if instead, you could:**
- Upload the logs and get an instant diagnosis
- See exactly which errors caused the outage  
- Get AI explanations in plain English
- Find the exact runbook or documentation to fix it
- All in under 30 seconds

That's LogSum. It uses **RAG (Retrieval-Augmented Generation)** to combine AI analysis with your team's knowledge base, so you get intelligent answers instead of just log parsing.

## Why LogSum?

- **Find problems fast** - Pattern detection highlights errors, timeouts, and anomalies
- **Get real answers** - AI explains what errors mean and suggests fixes  
- **Use your team's knowledge** - Connects errors to your documentation and runbooks
- **Works everywhere** - JSON, plaintext, or any log format

## Quick Start

### Install
```bash
go install github.com/yildizm/LogSum/cmd/logsum@latest
```

### Basic Usage
```bash
# Analyze any log file
logsum analyze /var/log/app.log

# Get AI insights
logsum analyze --ai /var/log/app.log

# Connect with your docs  
logsum analyze --ai --docs ./team-docs/ /var/log/app.log

# Monitor in real-time
logsum watch /var/log/app.log
```

## Real Examples

### The 3 AM Incident (Before LogSum)
```
2024-01-15 14:32:17 ERROR: Connection timeout to redis://cache:6379
2024-01-15 14:32:18 ERROR: Failed to process user session
2024-01-15 14:32:19 ERROR: Database query failed: connection refused
```
*You spend 2 hours googling, checking configs, restarting services... Users are angry, your manager is texting, and you still don't know the root cause.*

### With LogSum + RAG
```
🔍 Found 3 critical errors in /var/log/app.log

💡 AI Analysis:
   Redis connection timeout detected. This typically occurs when:
   - Redis server is down or unreachable
   - Network connectivity issues between app and cache
   - Redis maxclients limit exceeded

📚 Related Documentation:
   → docs/redis-troubleshooting.md
   → runbooks/cache-recovery.md

🛠️ Recommended Actions:
   1. Check Redis server status: redis-cli ping
   2. Verify network connectivity to cache:6379
   3. Review Redis connection pool settings
```

## Key Features

- **Smart Pattern Detection** - Automatically finds errors, timeouts, performance issues
- **AI-Powered Analysis** - Explains problems in plain English with actionable solutions
- **RAG Integration** - Combines AI with your team's docs for context-aware insights
- **Semantic Search** - Finds relevant documentation using vector similarity
- **Multiple Output Formats** - Text, JSON, Markdown for humans and automation
- **Real-time Monitoring** - Watch logs as they happen with live analysis
- **Performance Monitoring** - Built-in metrics to track LogSum's own performance
- **Fast & Lightweight** - Analyze 100K log entries in under 100ms

## Advanced Features

### AI Providers
LogSum works with multiple AI providers:

**Ollama (Recommended - Free & Local)**
```bash
# Install Ollama
curl -fsSL https://ollama.ai/install.sh | sh
ollama pull llama3.2
ollama serve

# Use with LogSum
logsum analyze --ai /var/log/app.log
```

**OpenAI**
```bash
export OPENAI_API_KEY="your-key-here"
logsum analyze --ai /var/log/app.log
```

### Configuration
```bash
# Generate config file
logsum config init

# Edit .logsum.yaml for your needs
logsum analyze --config .logsum.yaml /var/log/app.log
```

### RAG (Retrieval-Augmented Generation)
LogSum's RAG system combines AI with your team's knowledge:

```bash
# Link errors to docs (correlation only)
logsum analyze --correlate --docs ./team-docs/ /var/log/app.log

# Full RAG: AI + documentation correlation  
logsum analyze --ai --docs ./runbooks/ /var/log/app.log
```

**How RAG works:**
1. **Retrieval**: Finds relevant docs using semantic search
2. **Augmentation**: Provides context to the AI model
3. **Generation**: AI gives answers based on your specific documentation

This means instead of generic advice, you get solutions tailored to your team's processes and infrastructure.

## Output Formats

```bash
# Beautiful terminal output (default)
logsum analyze /var/log/app.log

# JSON for automation
logsum analyze --output json /var/log/app.log

# Markdown reports
logsum analyze --output markdown /var/log/app.log > report.md
```

## Performance

Built for speed and scale:

```
10K log entries:     3.3ms
100K log entries:    28ms  
1M log entries:      245ms
Vector search:       570μs
```

## Requirements

- Go 1.24+ (for installation)
- Optional: Ollama or OpenAI API key for AI features

## Installation Troubleshooting

**Command not found after install?**
```bash
# Add Go's bin directory to PATH
export PATH="$PATH:$(go env GOPATH)/bin"

# Make permanent
echo 'export PATH="$PATH:$(go env GOPATH)/bin"' >> ~/.bashrc
```

**Windows emoji issues?**
```bash
logsum --no-emoji analyze app.log
```

## Commands Reference

```bash
# Analysis
logsum analyze [file]              # Basic analysis
logsum analyze --ai [file]         # AI-powered analysis
logsum analyze --monitor [file]    # With performance monitoring

# Real-time
logsum watch [file]                # Monitor file changes

# Configuration  
logsum config init                 # Create config file
logsum config show                 # View current config

# Performance monitoring
logsum monitor start               # Start metrics collection
logsum monitor report              # Generate performance report

# Utility
logsum --version                   # Show version
logsum [command] --help            # Get help
```

## Development

Want to contribute? 

```bash
git clone https://github.com/yildizm/LogSum.git
cd LogSum
go install ./cmd/logsum
go test ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

MIT License - see [LICENSE](LICENSE) for details.

---

**Stop debugging in the dark. Start using LogSum.**

Get intelligent log analysis in seconds, not hours.