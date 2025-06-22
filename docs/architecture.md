# LogSum Architecture Diagram

## Overall System Architecture

```mermaid
graph TB
    %% CLI Layer
    CLI[CLI Interface<br/>internal/cli/]
    CLI --> |analyze command| ANALYZE[Analyze Handler<br/>analyze.go]
    CLI --> |monitor command| MONITOR[Monitor Handler<br/>monitor.go]
    CLI --> |watch command| WATCH[Watch Handler<br/>watch.go]

    %% Analysis Decision Logic
    ANALYZE --> DECISION{Flag Detection}
    DECISION --> |--ai (with/without --correlate)| AIONLY[AI Analysis + Correlation Summary<br/>performAIAnalysis()]
    DECISION --> |--correlate only| CORRONLY[Standard + Correlation<br/>runCLIAnalysis()]
    DECISION --> |no flags| STANDARD[Standard Analysis<br/>engine.Analyze()]
    
    %% Monitoring Integration (Optional)
    ANALYZE --> |--monitor flag| MONITOR_SETUP[Setup Monitoring<br/>setupMonitoring()]
    MONITOR_SETUP --> COLLECTOR[Metrics Collector<br/>internal/monitor/]
    COLLECTOR --> REALTIME[Real-time Display<br/>showRealTimeMetrics()]

    %% Core Analysis Pipeline
    AIONLY --> ENGINE[Analyzer Engine<br/>internal/analyzer/]
    AIONLY --> CORR[Correlator<br/>internal/correlation/]
    AIONLY --> AI[AI Analyzer<br/>AnalyzeWithAI()]
    CORR --> AI
    
    CORRONLY --> ENGINE
    CORRONLY --> CORR2[Correlator<br/>internal/correlation/]
    
    STANDARD --> ENGINE
    
    %% Monitoring Instrumentation (when --monitor enabled)
    COLLECTOR -.-> |Wraps Operations| ENGINE
    COLLECTOR -.-> |Tracks Performance| CORR
    COLLECTOR -.-> |Monitors AI Calls| AI
    REALTIME -.-> |Updates Every 2s| COLLECTOR

    %% Analysis Engine Components
    ENGINE --> PATTERN[Pattern Matcher<br/>engine.go]
    ENGINE --> INSIGHT[Insight Generator<br/>engine.go]
    ENGINE --> TIMELINE[Timeline Generator<br/>engine.go]

    %% AI Analysis Components
    AI --> |AI Providers| OLLAMA[Ollama Provider<br/>internal/ai/providers/ollama/]
    AI --> |AI Providers| OPENAI[OpenAI Provider<br/>internal/ai/providers/openai/]
    AI --> |Prompt Building| PROMPT[go-promptfmt<br/>External Library]

    %% Correlation System
    CORR --> |Document Search| DOCSTORE[Document Store<br/>internal/docstore/]
    CORR --> |Semantic Search| VECTOR[Vector Store<br/>internal/vectorstore/]
    CORR --> |Keyword Extraction| EXTRACT[Keyword Extractor<br/>extractor.go]
    CORR2 --> DOCSTORE
    CORR2 --> VECTOR
    CORR2 --> EXTRACT

    %% Document Store Components
    DOCSTORE --> MEMORY[Memory Store<br/>memory.go]
    DOCSTORE --> SCANNER[Document Scanner<br/>scanner.go]
    DOCSTORE --> INDEXER[Indexer<br/>indexer.go]

    %% Vector Store Components
    VECTOR --> TFIDF[TF-IDF Vectorizer<br/>vectorizer.go]
    VECTOR --> SIMILARITY[Similarity Calculator<br/>similarity.go]
    VECTOR --> CACHE[Vector Cache<br/>memory.go]

    %% Output Convergence
    AIONLY --> FORMAT[Formatters<br/>internal/formatter/]
    CORRONLY --> FORMAT
    STANDARD --> FORMAT

    %% Output Formatters
    FORMAT --> MARKDOWN[Markdown Formatter<br/>markdown.go]
    FORMAT --> JSON[JSON Formatter<br/>json.go]
    FORMAT --> TEXT[Text Formatter<br/>text.go]
    FORMAT --> CSV[CSV Formatter<br/>csv.go]

    %% Configuration & Utilities
    CONFIG[Config System<br/>internal/config/]
    LOGGER[Logging System<br/>internal/logger/]
    COMMON[Common Types<br/>internal/common/]
    MONITOR_SYS[Monitor System<br/>internal/monitor/]

    %% Data Flow
    CLI -.-> CONFIG
    ENGINE -.-> LOGGER
    AI -.-> LOGGER
    CORR -.-> LOGGER
    ENGINE -.-> COMMON
    AI -.-> COMMON
    CORR -.-> COMMON
    
    %% Monitoring Integration
    COLLECTOR -.-> MONITOR_SYS
    REALTIME -.-> MONITOR_SYS

    %% External Dependencies
    EXT1[External: go-logparser]
    EXT2[External: go-promptfmt] 
    EXT3[External: Cobra CLI]
    EXT4[External: Bubble Tea UI]

    CLI -.-> EXT3
    ANALYZE -.-> EXT4
    ENGINE -.-> EXT1
    AI -.-> EXT2

    %% Styling
    classDef cliLayer fill:#e1f5fe
    classDef decisionLayer fill:#fff3e0
    classDef pipelineLayer fill:#e8f5e8
    classDef analysisLayer fill:#f3e5f5
    classDef correlationLayer fill:#fff8e1
    classDef storageLayer fill:#fce4ec
    classDef outputLayer fill:#f1f8e9
    classDef monitorLayer fill:#e8f8f5

    class CLI,ANALYZE,MONITOR,WATCH cliLayer
    class DECISION decisionLayer
    class AIONLY,CORRONLY,STANDARD pipelineLayer
    class ENGINE,AI,AI2,PATTERN,INSIGHT,TIMELINE,OLLAMA,OPENAI,PROMPT analysisLayer
    class CORR,CORR2,EXTRACT correlationLayer
    class DOCSTORE,VECTOR,MEMORY,SCANNER,INDEXER,TFIDF,SIMILARITY,CACHE storageLayer
    class FORMAT,MARKDOWN,JSON,TEXT,CSV outputLayer
    class MONITOR_SETUP,COLLECTOR,REALTIME,MONITOR_SYS monitorLayer
```

## Data Flow Architecture

```mermaid
sequenceDiagram
    participant User
    participant CLI
    participant Engine
    participant AI
    participant Correlator
    participant DocStore
    participant VectorStore
    participant Formatter

    User->>CLI: logsum analyze --ai --docs
    
    Note over CLI,VectorStore: Setup Phase (Document Processing)
    CLI->>DocStore: Scan & index documents from /docs
    CLI->>VectorStore: Vectorize documents (TF-IDF)
    
    Note over CLI,Engine: Analysis Phase (Log Processing)
    CLI->>Engine: Parse logs & run base analysis
    Note over Engine: Extract patterns, timeline, insights<br/>NO vectorization of logs
    Engine-->>CLI: Base analysis results (patterns, errors, timeline)

    Note over CLI,Formatter: Correlation & AI Phase
    alt --ai flag (includes correlation summary)
        CLI->>Correlator: Correlate patterns & errors
        Note over Correlator: Extract keywords from log patterns/errors
        Correlator->>DocStore: Keyword search
        Correlator->>VectorStore: Vector search (keyword vectors vs doc vectors)
        DocStore-->>Correlator: Keyword matches
        VectorStore-->>Correlator: Similar documents
        Correlator-->>CLI: Correlation results
        CLI->>AI: Perform AI analysis with correlation context
        AI->>AI: Generate insights using correlation data
        AI-->>CLI: AI analysis + correlation summary
    else --correlate flag only
        CLI->>Correlator: Correlate patterns & errors
        Correlator->>DocStore: Keyword search
        Correlator->>VectorStore: Vector search
        DocStore-->>Correlator: Matched documents
        VectorStore-->>Correlator: Similar documents
        Correlator-->>CLI: Correlation results only
    end

    CLI->>Formatter: Format combined results
    Formatter-->>CLI: Formatted output
    CLI-->>User: Analysis report
```

## Component Interaction Map

```mermaid
graph LR
    %% Input Processing
    LOG[Log Files] --> PARSER[Parser]
    PARSER --> ENTRIES[Log Entries]
    DOCS[Documentation] --> SETUP[Document Setup]
    SETUP --> DOCSTORE[Document Store]
    SETUP --> VECTORS[Vector Store]

    %% Core Processing
    ENTRIES --> ENGINE[Analysis Engine]
    ENGINE --> PATTERNS[Pattern Detection]
    ENGINE --> INSIGHTS[Insight Generation]
    ENGINE --> TIMELINE[Timeline Analysis]

    %% Decision Point
    PATTERNS --> DECISION{Analysis Mode}
    DECISION --> |--ai| AI_PATH[AI Analysis + Correlation]
    DECISION --> |--correlate| CORR_PATH[Correlation Only]
    DECISION --> |no flags| DIRECT[Direct Output]

    %% AI Path (includes correlation)
    AI_PATH --> CORRELATOR[Correlator]
    CORRELATOR --> DOCSTORE
    CORRELATOR --> VECTORS
    CORRELATOR --> CORRELATIONS[Correlations]
    CORRELATIONS --> AI_ANALYZER[AI Analyzer]
    AI_ANALYZER --> LLM[LLM Providers]
    LLM --> AIRESULTS[AI Insights + Correlation Summary]

    %% Correlation-Only Path
    CORR_PATH --> CORRELATOR2[Correlator]
    CORRELATOR2 --> DOCSTORE
    CORRELATOR2 --> VECTORS
    CORRELATOR2 --> CORRRESULTS[Correlation Results]

    %% Output Generation
    PATTERNS --> OUTPUT[Analysis Results]
    INSIGHTS --> OUTPUT
    TIMELINE --> OUTPUT
    AIRESULTS --> OUTPUT
    CORRRESULTS --> OUTPUT

    OUTPUT --> FORMATTER[Formatters]
    FORMATTER --> REPORT[Final Report]

    %% Styling
    classDef input fill:#e3f2fd
    classDef processing fill:#f1f8e9
    classDef decision fill:#fff3e0
    classDef aiPath fill:#e8f5e8
    classDef ai fill:#fce4ec
    classDef correlation fill:#fff8e1
    classDef output fill:#f3e5f5

    class LOG,DOCS,ENTRIES,SETUP input
    class ENGINE,PARSER,PATTERNS,INSIGHTS,TIMELINE processing
    class DECISION decision
    class AI_PATH,CORR_PATH,DIRECT aiPath
    class AI_ANALYZER,LLM,AIRESULTS ai
    class DOCSTORE,VECTORS,CORRELATOR,CORRELATOR2,CORRELATIONS,CORRRESULTS correlation
    class OUTPUT,FORMATTER,REPORT output
```

## Package Dependency Graph

```mermaid
graph TD
    %% Main Entry Point
    MAIN[cmd/logsum/main.go]
    MAIN --> CLI[internal/cli/]

    %% CLI Dependencies
    CLI --> CONFIG[internal/config/]
    CLI --> ANALYZER[internal/analyzer/]
    CLI --> CORRELATION[internal/correlation/]
    CLI --> FORMATTER[internal/formatter/]
    CLI --> LOGGER[internal/logger/]
    CLI --> COMMON[internal/common/]
    CLI --> UI[internal/ui/]
    CLI --> MONITOR[internal/monitor/]

    %% Analyzer Dependencies
    ANALYZER --> AI[internal/ai/]
    ANALYZER --> COMMON
    ANALYZER --> LOGGER

    %% AI Dependencies
    AI --> PROVIDERS[internal/ai/providers/]
    AI --> COMMON
    PROVIDERS --> OLLAMA[ollama/]
    PROVIDERS --> OPENAI[openai/]

    %% Correlation Dependencies
    CORRELATION --> DOCSTORE[internal/docstore/]
    CORRELATION --> VECTORSTORE[internal/vectorstore/]
    CORRELATION --> LOGGER
    CORRELATION --> COMMON

    %% Document Store Dependencies
    DOCSTORE --> COMMON

    %% Vector Store Dependencies
    VECTORSTORE --> COMMON

    %% Formatter Dependencies
    FORMATTER --> COMMON

    %% UI Dependencies
    UI --> COMMON

    %% Monitor Dependencies
    MONITOR --> ANALYZER
    MONITOR --> COMMON

    %% Logger Dependencies (standalone)
    LOGGER --> [No internal dependencies]

    %% External Dependencies
    EXT_COBRA[External: github.com/spf13/cobra]
    EXT_BUBBLETEA[External: github.com/charmbracelet/bubbletea]
    EXT_LOGPARSER[External: go-logparser]
    EXT_PROMPTFMT[External: go-promptfmt]
    EXT_FSNOTIFY[External: github.com/fsnotify/fsnotify]

    CLI -.-> EXT_COBRA
    CLI -.-> EXT_FSNOTIFY
    UI -.-> EXT_BUBBLETEA
    ANALYZER -.-> EXT_LOGPARSER
    AI -.-> EXT_PROMPTFMT

    %% Styling
    classDef main fill:#ffebee
    classDef internal fill:#e8f5e8
    classDef utility fill:#fff3e0
    classDef external fill:#e1f5fe

    class MAIN main
    class CLI,CONFIG,ANALYZER,CORRELATION,FORMATTER,COMMON,AI,PROVIDERS,OLLAMA,OPENAI,DOCSTORE,VECTORSTORE,UI,MONITOR internal
    class LOGGER utility
    class EXT_COBRA,EXT_BUBBLETEA,EXT_LOGPARSER,EXT_PROMPTFMT,EXT_FSNOTIFY external
```

## Command Usage Patterns

```mermaid
graph TD
    %% Usage Scenarios
    USER[User Input] --> DETECTION{Flag Detection}
    
    %% Scenario 1: AI Analysis (always includes correlation summary)
    DETECTION --> |--ai| AI_ANALYSIS[AI Analysis + Correlation Summary]
    AI_ANALYSIS --> AI_INTERNAL[AI gets correlation context]
    AI_INTERNAL --> AI_OUTPUT[AI insights + correlation summary]
    
    %% Scenario 2: Correlation Only
    DETECTION --> |--correlate| CORR_ONLY[Correlation Only]
    CORR_ONLY --> CORR_STANDALONE[Standalone correlation analysis]
    CORR_STANDALONE --> CORR_OUTPUT[Document correlations]
    
    %% Scenario 3: Standard
    DETECTION --> |no flags| STANDARD[Standard Analysis]
    STANDARD --> BASIC_OUTPUT[Pattern analysis only]

    %% Use Cases
    AI_OUTPUT --> USE1[Development & Debugging<br/>Intelligent insights with transparency]
    CORR_OUTPUT --> USE2[Cost-conscious monitoring<br/>Compliance & audit]
    BASIC_OUTPUT --> USE3[Basic log processing<br/>Pattern detection]

    %% Styling
    classDef userInput fill:#e1f5fe
    classDef decision fill:#fff3e0
    classDef aiAnalysis fill:#e8f5e8
    classDef standard fill:#f3e5f5
    classDef output fill:#fce4ec
    classDef usecase fill:#f1f8e9

    class USER userInput
    class DETECTION decision
    class AI_ANALYSIS,AI_INTERNAL aiAnalysis
    class CORR_ONLY,STANDARD standard
    class AI_OUTPUT,CORR_OUTPUT,BASIC_OUTPUT output
    class USE1,USE2,USE3 usecase
```

## Component Hierarchy by Flag

```mermaid
graph TB
    %% Base Components (Always Present)
    subgraph BASE["Base Components (Always Active)"]
        direction TB
        RAW[Raw Log Parsing<br/>Entry extraction, timestamps]
        PATTERN[Pattern Matching<br/>Error detection, anomalies]
        RAW --> PATTERN
    end

    %% Correlation Layer
    subgraph CORR_LAYER["Correlation Layer (--correlate)"]
        direction TB
        DOC_SEARCH[Document Search<br/>Keyword matching]
        VECTOR_SEARCH[Vector Search<br/>Semantic similarity]
        HYBRID[Hybrid Scoring<br/>Combined relevance]
        
        DOC_SEARCH --> HYBRID
        VECTOR_SEARCH --> HYBRID
    end

    %% AI Layer
    subgraph AI_LAYER["AI Layer (--ai)"]
        direction TB
        CONTEXT[Context Building<br/>Document correlation]
        LLM[LLM Analysis<br/>Insights & recommendations]
        SYNTHESIS[Result Synthesis<br/>Human-readable output]
        
        CONTEXT --> LLM
        LLM --> SYNTHESIS
    end

    %% Monitoring Layer (Optional)
    subgraph MONITOR_LAYER["🔍 Monitoring Layer (--monitor)"]
        direction TB
        METRICS[Metrics Collection<br/>Performance tracking]
        DISPLAY[Real-time Display<br/>Live stats every 2s]
        EXPORT[Optional Export<br/>JSON metrics file]
        
        METRICS --> DISPLAY
        METRICS --> EXPORT
    end

    %% Flag Combinations
    subgraph FLAGS["Flag Combinations"]
        direction LR
        NO_FLAG[No Flags<br/>Basic Analysis]
        CORRELATE_FLAG[--correlate<br/>+ Document Correlation]
        AI_FLAG[--ai<br/>+ Document Correlation<br/>+ AI Analysis]
        MONITOR_FLAG[--monitor<br/>+ Real-time Performance<br/>+ Optional Export]
    end

    %% Dependencies
    BASE --> NO_FLAG
    BASE --> CORRELATE_FLAG
    BASE --> AI_FLAG
    
    CORR_LAYER --> CORRELATE_FLAG
    CORR_LAYER --> AI_FLAG
    AI_LAYER --> AI_FLAG
    
    %% Monitoring can be combined with any analysis
    MONITOR_LAYER -.-> NO_FLAG
    MONITOR_LAYER -.-> CORRELATE_FLAG
    MONITOR_LAYER -.-> AI_FLAG

    %% Output Types
    NO_FLAG --> OUT1[Pattern Analysis<br/>Error counts, timeline]
    CORRELATE_FLAG --> OUT2[Correlation Report<br/>Document matches, scores]
    AI_FLAG --> OUT3[AI Insights<br/>Root causes, recommendations<br/>+ Correlation transparency]
    MONITOR_FLAG --> OUT4[Real-time Metrics<br/>Performance monitoring<br/>Optional JSON export]

    %% Styling
    classDef baseComp fill:#e8f5e8,stroke:#4caf50
    classDef corrComp fill:#fff8e1,stroke:#ff9800
    classDef aiComp fill:#fce4ec,stroke:#e91e63
    classDef monitorComp fill:#e3f2fd,stroke:#2196f3
    classDef flagBox fill:#e3f2fd,stroke:#2196f3
    classDef output fill:#f3e5f5,stroke:#9c27b0

    class BASE,RAW,PATTERN baseComp
    class CORR_LAYER,DOC_SEARCH,VECTOR_SEARCH,HYBRID corrComp
    class AI_LAYER,CONTEXT,LLM,SYNTHESIS aiComp
    class MONITOR_LAYER,METRICS,DISPLAY,EXPORT monitorComp
    class FLAGS,NO_FLAG,CORRELATE_FLAG,AI_FLAG,MONITOR_FLAG flagBox
    class OUT1,OUT2,OUT3,OUT4 output
```