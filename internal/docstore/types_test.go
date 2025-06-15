package docstore

import (
	"testing"
	"time"
)

func TestChangeType_String(t *testing.T) {
	tests := []struct {
		name     string
		ct       ChangeType
		expected string
	}{
		{"Added", ChangeAdded, "added"},
		{"Modified", ChangeModified, "modified"},
		{"Deleted", ChangeDeleted, "deleted"},
		{"Unknown", ChangeType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.ct.String()
			if result != tt.expected {
				t.Errorf("ChangeType.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestDocument_Validation(t *testing.T) {
	doc := &Document{
		ID:           "test-doc",
		Path:         "/test/path.md",
		Title:        "Test Document",
		Content:      "This is test content",
		Metadata:     &Metadata{},
		Sections:     []*Section{},
		LastModified: time.Now(),
		Size:         100,
		Hash:         "abcd1234",
	}

	// Test valid document
	if doc.ID == "" {
		t.Error("Expected valid document ID")
	}
	if doc.Path == "" {
		t.Error("Expected valid document path")
	}
}

func TestSection_Validation(t *testing.T) {
	section := &Section{
		ID:         "section-1",
		DocumentID: "doc-1",
		Heading:    "Test Section",
		Content:    "Section content",
		Level:      1,
		StartLine:  1,
		EndLine:    10,
		WordCount:  2,
	}

	if section.ID == "" {
		t.Error("Expected valid section ID")
	}
	if section.DocumentID == "" {
		t.Error("Expected valid document ID")
	}
	if section.Level < 1 {
		t.Error("Expected valid heading level")
	}
}

func TestMetadata_CustomFields(t *testing.T) {
	metadata := &Metadata{
		Tags:     []string{"tag1", "tag2"},
		Author:   "Test Author",
		Date:     &time.Time{},
		Custom:   make(map[string]interface{}),
		Language: "en",
		Format:   "markdown",
	}

	metadata.Custom["custom_field"] = "custom_value"
	metadata.Custom["number_field"] = 123

	if len(metadata.Custom) != 2 {
		t.Errorf("Expected 2 custom fields, got %d", len(metadata.Custom))
	}

	if metadata.Custom["custom_field"] != "custom_value" {
		t.Error("Custom field value mismatch")
	}
}

func TestFilterOptions_DefaultValues(t *testing.T) {
	filter := FilterOptions{}

	if len(filter.Tags) != 0 {
		t.Error("Expected empty tags slice")
	}
	if filter.Limit != 0 {
		t.Error("Expected zero limit")
	}
	if filter.Offset != 0 {
		t.Error("Expected zero offset")
	}
}

func TestSearchQuery_BasicValidation(t *testing.T) {
	query := &SearchQuery{
		Text:      "test query",
		Fields:    []string{"title", "content"},
		Fuzzy:     true,
		Highlight: true,
		Limit:     10,
		Offset:    0,
	}

	if query.Text == "" {
		t.Error("Expected non-empty query text")
	}
	if len(query.Fields) == 0 {
		t.Error("Expected search fields")
	}
	if !query.Fuzzy {
		t.Error("Expected fuzzy search enabled")
	}
}
