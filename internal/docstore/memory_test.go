package docstore

import (
	"testing"
	"time"
)

func TestMemoryStore_AddAndGet(t *testing.T) {
	store := NewMemoryStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	doc := createTestDocument("test-1", "Test Document", "This is test content")

	// Test Add
	err := store.Add(doc)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Test Get
	retrieved, err := store.Get(doc.ID)
	if err != nil {
		t.Fatalf("Failed to get document: %v", err)
	}

	if retrieved.ID != doc.ID {
		t.Errorf("Expected ID %q, got %q", doc.ID, retrieved.ID)
	}
	if retrieved.Title != doc.Title {
		t.Errorf("Expected title %q, got %q", doc.Title, retrieved.Title)
	}
}

func TestMemoryStore_AddBatch(t *testing.T) {
	store := NewMemoryStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	docs := []*Document{
		createTestDocument("doc-1", "Document 1", "Content 1"),
		createTestDocument("doc-2", "Document 2", "Content 2"),
		createTestDocument("doc-3", "Document 3", "Content 3"),
	}

	err := store.AddBatch(docs)
	if err != nil {
		t.Fatalf("Failed to add batch: %v", err)
	}

	// Verify all documents were added
	for _, doc := range docs {
		retrieved, err := store.Get(doc.ID)
		if err != nil {
			t.Errorf("Failed to get document %s: %v", doc.ID, err)
		}
		if retrieved.ID != doc.ID {
			t.Errorf("Document ID mismatch: expected %q, got %q", doc.ID, retrieved.ID)
		}
	}
}

func TestMemoryStore_Update(t *testing.T) {
	store := NewMemoryStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	doc := createTestDocument("test-1", "Original Title", "Original content")

	// Add document
	err := store.Add(doc)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Update document
	doc.Title = "Updated Title"
	doc.Content = "Updated content"
	err = store.Update(doc)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Verify update
	retrieved, err := store.Get(doc.ID)
	if err != nil {
		t.Fatalf("Failed to get updated document: %v", err)
	}

	if retrieved.Title != "Updated Title" {
		t.Errorf("Expected updated title, got %q", retrieved.Title)
	}
	if retrieved.Content != "Updated content" {
		t.Errorf("Expected updated content, got %q", retrieved.Content)
	}
}

func TestMemoryStore_Delete(t *testing.T) {
	store := NewMemoryStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	doc := createTestDocument("test-1", "Test Document", "Test content")

	// Add document
	err := store.Add(doc)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Delete document
	err = store.Delete(doc.ID)
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Verify deletion
	_, err = store.Get(doc.ID)
	if err == nil {
		t.Error("Expected error when getting deleted document")
	}
}

func TestMemoryStore_List(t *testing.T) {
	store := NewMemoryStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Add test documents
	docs := []*Document{
		createTestDocumentWithMetadata("doc-1", "Doc 1", "Content 1", "author1", []string{"tag1", "tag2"}),
		createTestDocumentWithMetadata("doc-2", "Doc 2", "Content 2", "author2", []string{"tag2", "tag3"}),
		createTestDocumentWithMetadata("doc-3", "Doc 3", "Content 3", "author1", []string{"tag1"}),
	}

	for _, doc := range docs {
		if err := store.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	// Test list all
	allDocs, err := store.List(&FilterOptions{})
	if err != nil {
		t.Fatalf("Failed to list documents: %v", err)
	}
	if len(allDocs) != 3 {
		t.Errorf("Expected 3 documents, got %d", len(allDocs))
	}

	// Test filter by author
	authorFilter := &FilterOptions{Author: "author1"}
	authorDocs, err := store.List(authorFilter)
	if err != nil {
		t.Fatalf("Failed to list documents by author: %v", err)
	}
	if len(authorDocs) != 2 {
		t.Errorf("Expected 2 documents for author1, got %d", len(authorDocs))
	}

	// Test filter by tags
	tagFilter := &FilterOptions{Tags: []string{"tag1"}}
	tagDocs, err := store.List(tagFilter)
	if err != nil {
		t.Fatalf("Failed to list documents by tag: %v", err)
	}
	if len(tagDocs) != 2 {
		t.Errorf("Expected 2 documents with tag1, got %d", len(tagDocs))
	}

	// Test pagination
	pageFilter := &FilterOptions{Limit: 2, Offset: 1}
	pageDocs, err := store.List(pageFilter)
	if err != nil {
		t.Fatalf("Failed to list documents with pagination: %v", err)
	}
	if len(pageDocs) != 2 {
		t.Errorf("Expected 2 documents with pagination, got %d", len(pageDocs))
	}
}

func TestMemoryStore_Search(t *testing.T) {
	store := NewMemoryStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Add test documents
	docs := []*Document{
		createTestDocument("doc-1", "API Documentation", "This document describes the REST API endpoints"),
		createTestDocument("doc-2", "Setup Guide", "How to setup and configure the application"),
		createTestDocument("doc-3", "Performance Tips", "Optimization strategies for better performance"),
	}

	for _, doc := range docs {
		if err := store.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	// Test search
	query := &SearchQuery{
		Text:      "API",
		Highlight: true,
		Limit:     10,
	}

	results, err := store.Search(query)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	if len(results) == 0 {
		t.Error("Expected search results")
	}

	// Verify result contains API document
	found := false
	for _, result := range results {
		if result.Document.ID == "doc-1" {
			found = true
			if result.Score <= 0 {
				t.Error("Expected positive relevance score")
			}
			break
		}
	}
	if !found {
		t.Error("Expected to find API document in search results")
	}
}

func TestMemoryStore_Stats(t *testing.T) {
	store := NewMemoryStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Test empty store stats
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.DocumentCount != 0 {
		t.Errorf("Expected 0 documents, got %d", stats.DocumentCount)
	}

	// Add documents and test stats
	docs := []*Document{
		createTestDocument("doc-1", "Doc 1", "Content 1"),
		createTestDocument("doc-2", "Doc 2", "Content 2"),
	}

	for _, doc := range docs {
		if err := store.Add(doc); err != nil {
			t.Fatalf("Failed to add document: %v", err)
		}
	}

	stats, err = store.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats.DocumentCount != 2 {
		t.Errorf("Expected 2 documents, got %d", stats.DocumentCount)
	}
	if stats.TotalSize <= 0 {
		t.Error("Expected positive total size")
	}
	if stats.AverageDocSize <= 0 {
		t.Error("Expected positive average document size")
	}
}

func TestMemoryStore_Clear(t *testing.T) {
	store := NewMemoryStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	// Add documents
	doc := createTestDocument("test-1", "Test Document", "Test content")
	err := store.Add(doc)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Clear store
	err = store.Clear()
	if err != nil {
		t.Fatalf("Failed to clear store: %v", err)
	}

	// Verify store is empty
	stats, err := store.Stats()
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}
	if stats.DocumentCount != 0 {
		t.Errorf("Expected 0 documents after clear, got %d", stats.DocumentCount)
	}

	// Verify document is not retrievable
	_, err = store.Get(doc.ID)
	if err == nil {
		t.Error("Expected error when getting document from cleared store")
	}
}

func TestMemoryStore_ChangeListener(t *testing.T) {
	store := NewMemoryStore()
	defer func() {
		if err := store.Close(); err != nil {
			t.Errorf("Failed to close store: %v", err)
		}
	}()

	var receivedChanges []*DocumentChange
	listener := &testChangeListener{
		changes: &receivedChanges,
	}

	store.AddChangeListener(listener)

	// Add a document
	doc := createTestDocument("test-1", "Test Document", "Test content")
	err := store.Add(doc)
	if err != nil {
		t.Fatalf("Failed to add document: %v", err)
	}

	// Update the document
	doc.Title = "Updated Title"
	err = store.Update(doc)
	if err != nil {
		t.Fatalf("Failed to update document: %v", err)
	}

	// Delete the document
	err = store.Delete(doc.ID)
	if err != nil {
		t.Fatalf("Failed to delete document: %v", err)
	}

	// Verify change events
	if len(receivedChanges) != 3 {
		t.Errorf("Expected 3 change events, got %d", len(receivedChanges))
	}

	expectedTypes := []ChangeType{ChangeAdded, ChangeModified, ChangeDeleted}
	for i, change := range receivedChanges {
		if change.Type != expectedTypes[i] {
			t.Errorf("Change %d: expected type %v, got %v", i, expectedTypes[i], change.Type)
		}
		if change.DocumentID != doc.ID {
			t.Errorf("Change %d: expected document ID %q, got %q", i, doc.ID, change.DocumentID)
		}
	}
}

// Helper functions

func createTestDocument(id, title, content string) *Document {
	return &Document{
		ID:           id,
		Path:         "/test/" + id + ".md",
		Title:        title,
		Content:      content,
		Metadata:     &Metadata{Custom: make(map[string]interface{})},
		Sections:     []*Section{},
		LastModified: time.Now(),
		Size:         int64(len(content)),
		Hash:         "test-hash-" + id,
	}
}

func createTestDocumentWithMetadata(id, title, content, author string, tags []string) *Document {
	doc := createTestDocument(id, title, content)
	doc.Metadata.Author = author
	doc.Metadata.Tags = tags
	return doc
}

type testChangeListener struct {
	changes *[]*DocumentChange
}

func (tcl *testChangeListener) OnDocumentChanged(change *DocumentChange) {
	*tcl.changes = append(*tcl.changes, change)
}
