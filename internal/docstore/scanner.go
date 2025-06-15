package docstore

import (
	"crypto/sha256"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// MarkdownScanner implements the Scanner interface for markdown files
type MarkdownScanner struct {
	includePatterns []string
	excludePatterns []string
}

// NewMarkdownScanner creates a new markdown scanner
func NewMarkdownScanner() *MarkdownScanner {
	return &MarkdownScanner{
		includePatterns: []string{"*.md", "*.mdx", "*.markdown"},
		excludePatterns: []string{"node_modules", ".git", ".svn", "vendor"},
	}
}

// NewMarkdownScannerWithPatterns creates a scanner with custom patterns
func NewMarkdownScannerWithPatterns(include, exclude []string) *MarkdownScanner {
	scanner := NewMarkdownScanner()
	if len(include) > 0 {
		scanner.includePatterns = include
	}
	if len(exclude) > 0 {
		scanner.excludePatterns = exclude
	}
	return scanner
}

// ScanFile scans a single markdown file and returns a Document
func (ms *MarkdownScanner) ScanFile(path string) (*Document, error) {
	// Check if file exists
	info, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	// Read file content
	content, err := os.ReadFile(path) //nolint:gosec // File path comes from directory scanning
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	// Parse the document
	return ms.parseDocument(string(content), path, info)
}

// ScanDirectory scans a directory recursively for markdown files
func (ms *MarkdownScanner) ScanDirectory(path string, patterns []string) ([]*Document, error) {
	var documents []*Document

	// Use provided patterns or defaults
	includePatterns := patterns
	if len(includePatterns) == 0 {
		includePatterns = ms.includePatterns
	}

	err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if d.IsDir() {
			// Check if directory should be excluded
			dirName := d.Name()
			for _, excludePattern := range ms.excludePatterns {
				if matched, _ := filepath.Match(excludePattern, dirName); matched {
					return filepath.SkipDir
				}
			}
			// Skip hidden directories
			if strings.HasPrefix(dirName, ".") && dirName != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if file matches include patterns
		fileName := d.Name()
		matched := false
		for _, pattern := range includePatterns {
			if match, _ := filepath.Match(pattern, fileName); match {
				matched = true
				break
			}
		}

		if !matched {
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(fileName, ".") {
			return nil
		}

		// Scan the file
		doc, err := ms.ScanFile(filePath)
		if err != nil {
			// Log error but continue scanning
			fmt.Fprintf(os.Stderr, "Warning: Failed to scan file %s: %v\n", filePath, err)
			return nil
		}

		documents = append(documents, doc)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to scan directory %s: %w", path, err)
	}

	return documents, nil
}

// ParseContent parses content from a reader
func (ms *MarkdownScanner) ParseContent(reader io.Reader, path string) (*Document, error) {
	content, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read content: %w", err)
	}

	// Create mock file info
	info := &mockFileInfo{
		name:    filepath.Base(path),
		size:    int64(len(content)),
		modTime: time.Now(),
	}

	return ms.parseDocument(string(content), path, info)
}

// ExtractMetadata extracts YAML frontmatter from markdown content
func (ms *MarkdownScanner) ExtractMetadata(content string) (*Metadata, string, error) { //nolint:gocyclo // Complex YAML parsing logic
	// Check for YAML frontmatter
	if !strings.HasPrefix(content, "---") {
		return &Metadata{
			Custom: make(map[string]interface{}),
		}, content, nil
	}

	// Find the end of frontmatter
	lines := strings.Split(content, "\n")
	var frontmatterLines []string
	var contentLines []string
	inFrontmatter := false
	frontmatterEnded := false

	for i, line := range lines {
		if i == 0 && line == "---" {
			inFrontmatter = true
			continue
		}

		if inFrontmatter && line == "---" {
			frontmatterEnded = true
			inFrontmatter = false
			continue
		}

		if inFrontmatter {
			frontmatterLines = append(frontmatterLines, line)
		} else if frontmatterEnded || !inFrontmatter {
			contentLines = append(contentLines, line)
		}
	}

	// Parse YAML frontmatter
	metadata := &Metadata{
		Custom: make(map[string]interface{}),
	}

	if len(frontmatterLines) > 0 {
		frontmatterContent := strings.Join(frontmatterLines, "\n")
		var yamlData map[string]interface{}

		if err := yaml.Unmarshal([]byte(frontmatterContent), &yamlData); err != nil {
			return nil, "", fmt.Errorf("failed to parse YAML frontmatter: %w", err)
		}

		// Extract known fields
		if title, ok := yamlData["title"].(string); ok {
			metadata.Custom["title"] = title
		}
		if author, ok := yamlData["author"].(string); ok {
			metadata.Author = author
		}
		if dateStr, ok := yamlData["date"].(string); ok {
			if date, err := time.Parse("2006-01-02", dateStr); err == nil {
				metadata.Date = &date
			} else if date, err := time.Parse(time.RFC3339, dateStr); err == nil {
				metadata.Date = &date
			}
		}
		if tags, ok := yamlData["tags"].([]interface{}); ok {
			for _, tag := range tags {
				if tagStr, ok := tag.(string); ok {
					metadata.Tags = append(metadata.Tags, tagStr)
				}
			}
		}
		if lang, ok := yamlData["language"].(string); ok {
			metadata.Language = lang
		}

		// Store all other fields in custom
		for key, value := range yamlData {
			if key != "title" && key != "author" && key != "date" && key != "tags" && key != "language" {
				metadata.Custom[key] = value
			}
		}
	}

	remainingContent := strings.Join(contentLines, "\n")
	remainingContent = strings.TrimLeft(remainingContent, "\n")

	return metadata, remainingContent, nil
}

// SplitSections splits markdown content into logical sections
func (ms *MarkdownScanner) SplitSections(content string) ([]*Section, error) {
	var sections []*Section
	lines := strings.Split(content, "\n")

	var currentSection *Section
	var currentContent []string
	lineNumber := 1

	for i, line := range lines {
		// Check if line is a heading
		if headingLevel := ms.getHeadingLevel(line); headingLevel > 0 {
			// Save previous section if it exists
			if currentSection != nil {
				currentSection.Content = strings.Join(currentContent, "\n")
				currentSection.EndLine = i
				currentSection.WordCount = ms.countWords(currentSection.Content)
				sections = append(sections, currentSection)
			}

			// Start new section
			headingText := ms.extractHeadingText(line)
			currentSection = &Section{
				ID:        fmt.Sprintf("section_%d", len(sections)+1),
				Heading:   headingText,
				Level:     headingLevel,
				StartLine: i + 1,
			}
			currentContent = []string{}
		} else {
			// Add line to current section content
			currentContent = append(currentContent, line)
		}
		lineNumber++
	}

	// Save the last section
	if currentSection != nil {
		currentSection.Content = strings.Join(currentContent, "\n")
		currentSection.EndLine = len(lines)
		currentSection.WordCount = ms.countWords(currentSection.Content)
		sections = append(sections, currentSection)
	}

	// If no sections were found, create a single section with all content
	if len(sections) == 0 {
		sections = []*Section{{
			ID:        "section_1",
			Heading:   "Content",
			Content:   content,
			Level:     1,
			StartLine: 1,
			EndLine:   len(lines),
			WordCount: ms.countWords(content),
		}}
	}

	return sections, nil
}

// Helper methods

func (ms *MarkdownScanner) parseDocument(content, path string, info os.FileInfo) (*Document, error) {
	// Extract metadata and clean content
	metadata, cleanContent, err := ms.ExtractMetadata(content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract metadata: %w", err)
	}

	// Extract title
	title := ms.extractTitle(cleanContent, metadata)

	// Split into sections
	sections, err := ms.SplitSections(cleanContent)
	if err != nil {
		return nil, fmt.Errorf("failed to split sections: %w", err)
	}

	// Generate document ID and hash
	docID := ms.generateDocumentID(path)
	hash := ms.generateHash(content)

	// Set metadata format
	metadata.Format = "markdown"

	// Create document
	doc := &Document{
		ID:           docID,
		Path:         path,
		Title:        title,
		Content:      cleanContent,
		Metadata:     metadata,
		Sections:     sections,
		LastModified: info.ModTime(),
		Size:         info.Size(),
		Hash:         hash,
	}

	// Set document ID in sections
	for _, section := range sections {
		section.DocumentID = docID
	}

	return doc, nil
}

func (ms *MarkdownScanner) extractTitle(content string, metadata *Metadata) string {
	// First check metadata for title
	if title, ok := metadata.Custom["title"].(string); ok && title != "" {
		return title
	}

	// Look for first H1 heading
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
		// Also check for underlined headings
		if line != "" && len(lines) > 1 {
			nextLineIdx := 0
			for i, l := range lines {
				if l == line {
					if i+1 < len(lines) {
						nextLineIdx = i + 1
						break
					}
				}
			}
			if nextLineIdx > 0 && strings.TrimSpace(lines[nextLineIdx]) != "" {
				nextLine := lines[nextLineIdx]
				if strings.HasPrefix(nextLine, "===") || strings.HasPrefix(nextLine, "---") {
					return strings.TrimSpace(line)
				}
			}
		}
	}

	// Fall back to filename
	return strings.TrimSuffix(filepath.Base(content), filepath.Ext(content))
}

func (ms *MarkdownScanner) getHeadingLevel(line string) int {
	trimmed := strings.TrimSpace(line)

	// ATX headings (# ## ###)
	if strings.HasPrefix(trimmed, "#") {
		count := 0
		for _, char := range trimmed {
			switch char {
			case '#':
				count++
			case ' ':
				if count <= 6 && count > 0 {
					return count
				}
				return 0
			default:
				return 0 // Invalid heading
			}
		}
		if count <= 6 && count > 0 {
			return count
		}
	}

	return 0
}

func (ms *MarkdownScanner) extractHeadingText(line string) string {
	trimmed := strings.TrimSpace(line)

	// Remove # characters and leading/trailing spaces
	text := strings.TrimLeft(trimmed, "#")
	text = strings.TrimSpace(text)

	return text
}

func (ms *MarkdownScanner) countWords(text string) int {
	// Simple word counting
	fields := strings.Fields(text)
	return len(fields)
}

func (ms *MarkdownScanner) generateDocumentID(path string) string {
	// Use file path as base for ID, but make it safe
	id := strings.ReplaceAll(path, "/", "_")
	id = strings.ReplaceAll(id, "\\", "_")
	id = strings.ReplaceAll(id, " ", "_")

	// Remove extension
	if ext := filepath.Ext(id); ext != "" {
		id = id[:len(id)-len(ext)]
	}

	return id
}

func (ms *MarkdownScanner) generateHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf("%x", hash)
}

// mockFileInfo implements os.FileInfo for testing
type mockFileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (m *mockFileInfo) Name() string       { return m.name }
func (m *mockFileInfo) Size() int64        { return m.size }
func (m *mockFileInfo) Mode() os.FileMode  { return 0o644 }
func (m *mockFileInfo) ModTime() time.Time { return m.modTime }
func (m *mockFileInfo) IsDir() bool        { return false }
func (m *mockFileInfo) Sys() interface{}   { return nil }
