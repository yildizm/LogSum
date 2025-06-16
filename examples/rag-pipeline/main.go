package main

import (
	"context"
	"fmt"
	"log"

	"github.com/yildizm/LogSum/internal/common"
	"github.com/yildizm/LogSum/internal/correlation"
	"github.com/yildizm/LogSum/internal/docstore"
	"github.com/yildizm/LogSum/internal/vectorstore"
	"github.com/yildizm/go-logparser"
)

// RAGPipelineExample demonstrates the enhanced RAG pipeline that combines
// keyword search with semantic vector search for improved document correlation
//
//nolint:gocyclo // Demo function intentionally shows multiple features
func main() {
	fmt.Println("=== Enhanced RAG Pipeline Demo ===")
	fmt.Println("Demonstrating hybrid search: Keywords + Vector Similarity")
	fmt.Println()

	ctx := context.Background()

	// 1. Setup Document Store
	fmt.Println("1. Setting up document store...")
	docStore := docstore.NewMemoryStore()

	// Add sample documentation
	docs := []*docstore.Document{
		{
			ID:      "database-timeout",
			Path:    "/docs/troubleshooting/database-timeout.md",
			Title:   "Database Connection Timeout Troubleshooting",
			Content: "When experiencing database connection timeouts, check network connectivity, firewall settings, and database server load. Common causes include network latency, connection pool exhaustion, and server overload.",
		},
		{
			ID:      "postgres-connection",
			Path:    "/docs/database/postgresql-connection.md",
			Title:   "PostgreSQL Connection Guide",
			Content: "PostgreSQL connection issues can arise from misconfigured connection strings, authentication failures, or server unavailability. Verify connection parameters and server status.",
		},
		{
			ID:      "network-issues",
			Path:    "/docs/infrastructure/network-troubleshooting.md",
			Title:   "Network Troubleshooting Guide",
			Content: "Network connectivity problems can manifest as timeouts, connection refused errors, or DNS resolution failures. Check network configuration, DNS settings, and firewall rules.",
		},
		{
			ID:      "auth-failures",
			Path:    "/docs/security/authentication-troubleshooting.md",
			Title:   "Authentication Failure Resolution",
			Content: "Authentication failures occur due to invalid credentials, expired tokens, or misconfigured authentication providers. Verify user credentials and authentication configuration.",
		},
		{
			ID:      "ssl-certificate",
			Path:    "/docs/security/ssl-certificate-issues.md",
			Title:   "SSL Certificate Problems",
			Content: "SSL certificate verification failures can prevent secure connections. Common issues include expired certificates, hostname mismatches, and untrusted certificate authorities.",
		},
		{
			ID:      "performance-monitoring",
			Path:    "/docs/monitoring/performance-monitoring.md",
			Title:   "Application Performance Monitoring",
			Content: "Monitor application performance using metrics like response time, throughput, and error rates. Set up alerts for performance degradation and establish baseline measurements.",
		},
	}

	for _, doc := range docs {
		if err := docStore.Add(doc); err != nil {
			log.Printf("Failed to add document %s: %v", doc.ID, err)
		}
	}

	fmt.Printf("Added %d documents to document store\n", len(docs))

	// 2. Setup Vector Store and Vectorizer
	fmt.Println("\n2. Setting up vector store for semantic search...")
	vectorizer := vectorstore.NewTFIDFVectorizer(500)
	vectorStore := vectorstore.NewMemoryStore(
		vectorstore.WithMaxVectors(1000),
		vectorstore.WithNormalization(),
	)
	defer func() {
		if err := vectorStore.Close(); err != nil {
			log.Printf("Failed to close vector store: %v", err)
		}
	}()

	// 3. Setup Enhanced Correlator
	fmt.Println("\n3. Setting up enhanced correlator with hybrid search...")
	correlator := correlation.NewCorrelator()

	// Configure document store
	if err := correlator.SetDocumentStore(docStore); err != nil {
		log.Printf("Failed to set document store: %v", err)
		return
	}

	// Configure vector store for semantic search
	if err := correlator.SetVectorStore(vectorStore, vectorizer); err != nil {
		log.Printf("Failed to set vector store: %v", err)
		return
	}

	// Configure hybrid search weights
	hybridConfig := &correlation.HybridSearchConfig{
		KeywordWeight:  0.6, // Favor keyword matching slightly
		VectorWeight:   0.4, // But also consider semantic similarity
		MaxResults:     5,
		VectorTopK:     8,
		MinVectorScore: 0.1,
		EnableVector:   true,
	}

	if err := correlator.SetHybridSearchConfig(hybridConfig); err != nil {
		log.Printf("Failed to set hybrid config: %v", err)
		return
	}

	// Index documents for vector search
	fmt.Println("\n4. Indexing documents for vector search...")
	if err := correlator.IndexDocuments(ctx); err != nil {
		log.Printf("Failed to index documents: %v", err)
		return
	}

	fmt.Printf("Vectorizer trained with %d vocabulary terms\n", vectorizer.GetVocabularySize())
	fmt.Printf("Vector store contains %d documents\n", vectorStore.Size())

	// 5. Create sample error analysis
	fmt.Println("\n5. Analyzing sample error patterns...")

	testCases := []struct {
		name     string
		pattern  *common.Pattern
		logEntry *common.LogEntry
	}{
		{
			name: "Database Connection Timeout",
			pattern: &common.Pattern{
				Name:        "DatabaseConnectionTimeout",
				Description: "Database connection timeout error",
				Regex:       `connection.*timeout.*database`,
			},
			logEntry: &common.LogEntry{
				LogEntry: logparser.LogEntry{
					Message: "connection timeout while connecting to database server",
				},
			},
		},
		{
			name: "SSL Certificate Error",
			pattern: &common.Pattern{
				Name:        "SSLCertificateError",
				Description: "SSL certificate verification failure",
				Regex:       `ssl.*certificate.*error`,
			},
			logEntry: &common.LogEntry{
				LogEntry: logparser.LogEntry{
					Message: "SSL certificate verification failed for hostname",
				},
			},
		},
		{
			name: "Authentication Failure",
			pattern: &common.Pattern{
				Name:        "AuthenticationFailure",
				Description: "Authentication failed due to invalid credentials",
				Regex:       `authentication.*failed`,
			},
			logEntry: &common.LogEntry{
				LogEntry: logparser.LogEntry{
					Message: "authentication failed: invalid username or password",
				},
			},
		},
	}

	for i, testCase := range testCases {
		fmt.Printf("\n--- Test Case %d: %s ---\n", i+1, testCase.name)

		analysis := &common.Analysis{
			Patterns: []common.PatternMatch{
				{
					Pattern: testCase.pattern,
					Matches: []*common.LogEntry{testCase.logEntry},
					Count:   1,
				},
			},
		}

		// Perform correlation using hybrid search
		result, err := correlator.Correlate(ctx, analysis)
		if err != nil {
			log.Printf("Correlation failed for %s: %v", testCase.name, err)
			continue
		}

		// Display results
		fmt.Printf("Total patterns analyzed: %d\n", result.TotalPatterns)
		fmt.Printf("Patterns with correlations: %d\n", result.CorrelatedPatterns)

		if len(result.Correlations) > 0 {
			corr := result.Correlations[0]
			fmt.Printf("Pattern: %s\n", corr.Pattern.Name)
			fmt.Printf("Keywords extracted: %v\n", corr.Keywords)
			fmt.Printf("Document matches found: %d\n", len(corr.DocumentMatches))

			for j, match := range corr.DocumentMatches {
				fmt.Printf("\n  Match %d:\n", j+1)
				fmt.Printf("    Document: %s\n", match.Document.Title)
				fmt.Printf("    Search Method: %s\n", match.SearchMethod)
				fmt.Printf("    Combined Score: %.3f\n", match.Score)
				fmt.Printf("    Keyword Score: %.3f\n", match.KeywordScore)
				fmt.Printf("    Vector Score: %.3f\n", match.VectorScore)
				if len(match.MatchedKeywords) > 0 {
					fmt.Printf("    Matched Keywords: %v\n", match.MatchedKeywords)
				}
			}
		} else {
			fmt.Println("No correlations found")
		}
	}

	// 6. Compare keyword-only vs hybrid search
	fmt.Println("\n=== Comparison: Keyword-Only vs Hybrid Search ===")

	// Disable vector search temporarily
	keywordOnlyConfig := &correlation.HybridSearchConfig{
		KeywordWeight:  1.0,
		VectorWeight:   0.0,
		MaxResults:     5,
		VectorTopK:     8,
		MinVectorScore: 0.1,
		EnableVector:   false,
	}

	if err := correlator.SetHybridSearchConfig(keywordOnlyConfig); err != nil {
		log.Printf("Failed to set keyword-only config: %v", err)
	}

	// Test with semantic query that might not match keywords exactly
	semanticTest := &common.Analysis{
		Patterns: []common.PatternMatch{
			{
				Pattern: &common.Pattern{
					Name:        "NetworkConnectivityIssue",
					Description: "Network connectivity problem causing service disruption",
					Regex:       `network.*connectivity.*issue`,
				},
				Matches: []*common.LogEntry{
					{
						LogEntry: logparser.LogEntry{
							Message: "service unavailable due to network connectivity issue",
						},
					},
				},
				Count: 1,
			},
		},
	}

	fmt.Println("\nKeyword-only search results:")
	keywordResult, err := correlator.Correlate(ctx, semanticTest)
	if err == nil && len(keywordResult.Correlations) > 0 {
		fmt.Printf("Found %d document matches\n", len(keywordResult.Correlations[0].DocumentMatches))
		for _, match := range keywordResult.Correlations[0].DocumentMatches {
			fmt.Printf("  - %s (Score: %.3f)\n", match.Document.Title, match.Score)
		}
	} else {
		fmt.Println("  No matches found")
	}

	// Re-enable hybrid search
	if err := correlator.SetHybridSearchConfig(hybridConfig); err != nil {
		log.Printf("Failed to re-enable hybrid config: %v", err)
	}

	fmt.Println("\nHybrid search results:")
	hybridResult, err := correlator.Correlate(ctx, semanticTest)
	if err == nil && len(hybridResult.Correlations) > 0 {
		fmt.Printf("Found %d document matches\n", len(hybridResult.Correlations[0].DocumentMatches))
		for _, match := range hybridResult.Correlations[0].DocumentMatches {
			fmt.Printf("  - %s (Score: %.3f, Method: %s)\n", match.Document.Title, match.Score, match.SearchMethod)
		}
	} else {
		fmt.Println("  No matches found")
	}

	// 7. Performance summary
	fmt.Println("\n=== Performance Summary ===")
	fmt.Printf("Vector store size: %d documents\n", vectorStore.Size())
	fmt.Printf("Vector dimension: %d\n", vectorizer.Dimension())
	fmt.Printf("Vocabulary size: %d terms\n", vectorizer.GetVocabularySize())

	stats, err := docStore.Stats()
	if err == nil {
		fmt.Printf("Document store: %d documents indexed\n", stats.DocumentCount)
	}

	fmt.Println("\n=== RAG Pipeline Demo Complete ===")
	fmt.Println("The enhanced correlation system now combines:")
	fmt.Println("  ✓ Traditional keyword matching for exact matches")
	fmt.Println("  ✓ Semantic vector search for conceptual similarity")
	fmt.Println("  ✓ Hybrid scoring to balance both approaches")
	fmt.Println("  ✓ Improved recall for complex error scenarios")
}
