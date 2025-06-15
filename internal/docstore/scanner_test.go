package docstore

import (
	"strings"
	"testing"
)

func TestMarkdownScanner_ExtractMetadata(t *testing.T) {
	scanner := NewMarkdownScanner()

	tests := []struct {
		name           string
		content        string
		expectedAuthor string
		expectedTags   int
		expectError    bool
	}{
		{
			name: "Valid frontmatter",
			content: `---
title: "Test Document"
author: "John Doe"
date: "2024-01-15"
tags: ["test", "example"]
---

# Test Content`,
			expectedAuthor: "John Doe",
			expectedTags:   2,
			expectError:    false,
		},
		{
			name: "No frontmatter",
			content: `# Test Document

This is content without frontmatter.`,
			expectedAuthor: "",
			expectedTags:   0,
			expectError:    false,
		},
		{
			name: "Invalid YAML",
			content: `---
title: "Test Document
author: John Doe
invalid: yaml: content
---

Content`,
			expectedAuthor: "",
			expectedTags:   0,
			expectError:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata, content, err := scanner.ExtractMetadata(tt.content)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError {
				if metadata.Author != tt.expectedAuthor {
					t.Errorf("Expected author %q, got %q", tt.expectedAuthor, metadata.Author)
				}
				if len(metadata.Tags) != tt.expectedTags {
					t.Errorf("Expected %d tags, got %d", tt.expectedTags, len(metadata.Tags))
				}
				if content == "" {
					t.Error("Expected non-empty content")
				}
			}
		})
	}
}

func TestMarkdownScanner_SplitSections(t *testing.T) {
	scanner := NewMarkdownScanner()

	content := `# Introduction

This is the introduction section.

## Getting Started

This is the getting started section.

### Prerequisites

You need these prerequisites.

## Advanced Topics

This covers advanced topics.`

	sections, err := scanner.SplitSections(content)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedSections := 4 // Introduction, Getting Started, Prerequisites, Advanced Topics
	if len(sections) != expectedSections {
		t.Errorf("Expected %d sections, got %d", expectedSections, len(sections))
	}

	// Test section levels
	expectedLevels := []int{1, 2, 3, 2}
	for i, section := range sections {
		if section.Level != expectedLevels[i] {
			t.Errorf("Section %d: expected level %d, got %d", i, expectedLevels[i], section.Level)
		}
	}

	// Test section content
	if !strings.Contains(sections[0].Content, "introduction section") {
		t.Error("First section should contain introduction content")
	}
}

func TestMarkdownScanner_GetHeadingLevel(t *testing.T) {
	scanner := NewMarkdownScanner()

	tests := []struct {
		line     string
		expected int
	}{
		{"# Heading 1", 1},
		{"## Heading 2", 2},
		{"### Heading 3", 3},
		{"#### Heading 4", 4},
		{"##### Heading 5", 5},
		{"###### Heading 6", 6},
		{"####### Too many", 0}, // Invalid
		{"Not a heading", 0},
		{"#No space", 0}, // Invalid
		{"", 0},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := scanner.getHeadingLevel(tt.line)
			if result != tt.expected {
				t.Errorf("getHeadingLevel(%q) = %d, want %d", tt.line, result, tt.expected)
			}
		})
	}
}

func TestMarkdownScanner_ExtractHeadingText(t *testing.T) {
	scanner := NewMarkdownScanner()

	tests := []struct {
		line     string
		expected string
	}{
		{"# Heading 1", "Heading 1"},
		{"## Heading 2", "Heading 2"},
		{"### Heading with spaces   ", "Heading with spaces"},
		{"#### ", ""},
		{"##### Multiple   Words", "Multiple   Words"},
	}

	for _, tt := range tests {
		t.Run(tt.line, func(t *testing.T) {
			result := scanner.extractHeadingText(tt.line)
			if result != tt.expected {
				t.Errorf("extractHeadingText(%q) = %q, want %q", tt.line, result, tt.expected)
			}
		})
	}
}

func TestMarkdownScanner_CountWords(t *testing.T) {
	scanner := NewMarkdownScanner()

	tests := []struct {
		text     string
		expected int
	}{
		{"hello world", 2},
		{"  hello   world  ", 2},
		{"", 0},
		{"single", 1},
		{"one two three four five", 5},
		{"punctuation, counts! as? words.", 4},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			result := scanner.countWords(tt.text)
			if result != tt.expected {
				t.Errorf("countWords(%q) = %d, want %d", tt.text, result, tt.expected)
			}
		})
	}
}

func TestMarkdownScanner_GenerateDocumentID(t *testing.T) {
	scanner := NewMarkdownScanner()

	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/file.md", "_path_to_file"},
		{"C:\\Windows\\file.md", "C:_Windows_file"},
		{"simple.md", "simple"},
		{"file with spaces.md", "file_with_spaces"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := scanner.generateDocumentID(tt.path)
			if result != tt.expected {
				t.Errorf("generateDocumentID(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestMarkdownScanner_ParseContent(t *testing.T) {
	scanner := NewMarkdownScanner()

	content := `---
title: "Test Document"
author: "Test Author"
tags: ["test"]
---

# Test Document

This is a test document with multiple sections.

## Section 1

Content for section 1.

## Section 2

Content for section 2.`

	reader := strings.NewReader(content)
	doc, err := scanner.ParseContent(reader, "/test/doc.md")

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if doc.Title != "Test Document" {
		t.Errorf("Expected title 'Test Document', got %q", doc.Title)
	}

	if doc.Metadata.Author != "Test Author" {
		t.Errorf("Expected author 'Test Author', got %q", doc.Metadata.Author)
	}

	if len(doc.Sections) != 3 { // Main content + 2 sections
		t.Errorf("Expected 3 sections, got %d", len(doc.Sections))
	}

	// Test that all sections have the correct document ID
	for _, section := range doc.Sections {
		if section.DocumentID != doc.ID {
			t.Errorf("Section document ID %q doesn't match document ID %q", section.DocumentID, doc.ID)
		}
	}
}

func TestMarkdownScanner_ExtractTitle(t *testing.T) {
	scanner := NewMarkdownScanner()

	tests := []struct {
		name     string
		content  string
		metadata *Metadata
		expected string
	}{
		{
			name:    "Title from metadata",
			content: "Some content",
			metadata: &Metadata{
				Custom: map[string]interface{}{
					"title": "Metadata Title",
				},
			},
			expected: "Metadata Title",
		},
		{
			name:     "Title from H1 heading",
			content:  "# Heading Title\n\nContent here",
			metadata: &Metadata{Custom: make(map[string]interface{})},
			expected: "Heading Title",
		},
		{
			name:     "No title found",
			content:  "Just some content without headings",
			metadata: &Metadata{Custom: make(map[string]interface{})},
			expected: "Just some content without headings", // Falls back to filename logic
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scanner.extractTitle(tt.content, tt.metadata)
			if !strings.Contains(result, strings.Split(tt.expected, " ")[0]) {
				t.Errorf("extractTitle() result %q should contain %q", result, tt.expected)
			}
		})
	}
}
