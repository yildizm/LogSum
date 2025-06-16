package vectorstore

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

// NewTFIDFVectorizer creates a new TF-IDF vectorizer with specified dimensions
func NewTFIDFVectorizer(dimensions int) *TFIDFVectorizer {
	return &TFIDFVectorizer{
		dimensions:    dimensions,
		vocabulary:    make(map[string]int),
		minWordLength: 2,
		maxWordLength: 50,
		stopWords:     getDefaultStopWords(),
	}
}

// Dimension returns the vector dimension
func (v *TFIDFVectorizer) Dimension() int {
	return v.dimensions
}

// Fit trains the vectorizer on a corpus of documents
func (v *TFIDFVectorizer) Fit(documents []string) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	if len(documents) == 0 {
		return fmt.Errorf("cannot fit on empty document corpus")
	}

	// Reset state
	v.vocabulary = make(map[string]int)
	v.idf = nil
	v.documentCount = len(documents)
	v.fitted = false

	// Build vocabulary and calculate document frequencies
	wordDocCounts := make(map[string]int)

	for _, doc := range documents {
		words := v.tokenize(doc)
		uniqueWords := make(map[string]bool)

		for _, word := range words {
			if !v.isValidWord(word) {
				continue
			}
			uniqueWords[word] = true
		}

		for word := range uniqueWords {
			wordDocCounts[word]++
		}
	}

	// Select top dimensions words by document frequency
	type wordFreq struct {
		word  string
		count int
	}

	wordFreqs := make([]wordFreq, 0, len(wordDocCounts))
	for word, count := range wordDocCounts {
		wordFreqs = append(wordFreqs, wordFreq{word: word, count: count})
	}

	// Sort by frequency (descending) to get most common words
	sort.Slice(wordFreqs, func(i, j int) bool {
		return wordFreqs[i].count > wordFreqs[j].count
	})

	// Build vocabulary with top words
	vocabSize := v.dimensions
	if len(wordFreqs) < vocabSize {
		vocabSize = len(wordFreqs)
	}

	for i := 0; i < vocabSize; i++ {
		word := wordFreqs[i].word
		v.vocabulary[word] = i
	}

	// Calculate IDF values
	v.idf = make([]float32, len(v.vocabulary))
	for word, index := range v.vocabulary {
		docCount := wordDocCounts[word]
		idfValue := float32(math.Log(float64(v.documentCount) / float64(docCount)))
		v.idf[index] = idfValue
	}

	v.fitted = true
	return nil
}

// FitTransform fits the vectorizer and transforms the documents
func (v *TFIDFVectorizer) FitTransform(documents []string) ([][]float32, error) {
	if err := v.Fit(documents); err != nil {
		return nil, err
	}

	vectors := make([][]float32, len(documents))
	for i, doc := range documents {
		vector, err := v.Vectorize(doc)
		if err != nil {
			return nil, fmt.Errorf("failed to vectorize document %d: %w", i, err)
		}
		vectors[i] = vector
	}

	return vectors, nil
}

// Vectorize converts text to a TF-IDF vector
func (v *TFIDFVectorizer) Vectorize(text string) ([]float32, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.fitted {
		return nil, fmt.Errorf("vectorizer must be fitted before vectorizing")
	}

	// Initialize vector with zeros
	vector := make([]float32, v.dimensions)

	// Tokenize and count word frequencies
	words := v.tokenize(text)
	wordCounts := make(map[string]int)
	totalWords := 0

	for _, word := range words {
		if !v.isValidWord(word) {
			continue
		}
		wordCounts[word]++
		totalWords++
	}

	if totalWords == 0 {
		return vector, nil
	}

	// Calculate TF-IDF for each word in vocabulary
	for word, count := range wordCounts {
		if index, exists := v.vocabulary[word]; exists {
			tf := float32(count) / float32(totalWords)
			tfidf := tf * v.idf[index]
			vector[index] = tfidf
		}
	}

	return vector, nil
}

// tokenize splits text into words
func (v *TFIDFVectorizer) tokenize(text string) []string {
	// Convert to lowercase
	text = strings.ToLower(text)

	// Remove punctuation and split on whitespace
	re := regexp.MustCompile(`[^\p{L}\p{N}]+`)
	text = re.ReplaceAllString(text, " ")

	// Split into words
	words := strings.Fields(text)

	return words
}

// isValidWord checks if a word should be included in the vocabulary
func (v *TFIDFVectorizer) isValidWord(word string) bool {
	// Check length constraints
	if len(word) < v.minWordLength || len(word) > v.maxWordLength {
		return false
	}

	// Check if it's a stop word
	if v.stopWords[word] {
		return false
	}

	// Check if it's all numbers
	if regexp.MustCompile(`^\d+$`).MatchString(word) {
		return false
	}

	return true
}

// getDefaultStopWords returns a set of common English stop words
func getDefaultStopWords() map[string]bool {
	stopWords := []string{
		"a", "an", "and", "are", "as", "at", "be", "been", "by", "for", "from",
		"has", "he", "in", "is", "it", "its", "of", "on", "that", "the", "to",
		"was", "will", "with", "the", "this", "but", "they", "have", "had",
		"what", "said", "each", "which", "she", "do", "how", "their", "if",
		"up", "out", "many", "then", "them", "these", "so", "some", "her",
		"would", "make", "like", "him", "into", "time", "two", "more", "go",
		"no", "way", "could", "my", "than", "first", "been", "call", "who",
		"oil", "sit", "now", "find", "down", "day", "did", "get", "come",
		"made", "may", "part",
	}

	stopWordSet := make(map[string]bool)
	for _, word := range stopWords {
		stopWordSet[word] = true
	}

	return stopWordSet
}

// SetMinWordLength sets the minimum word length for vocabulary
func (v *TFIDFVectorizer) SetMinWordLength(length int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.minWordLength = length
}

// SetMaxWordLength sets the maximum word length for vocabulary
func (v *TFIDFVectorizer) SetMaxWordLength(length int) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.maxWordLength = length
}

// AddStopWords adds additional stop words to filter out
func (v *TFIDFVectorizer) AddStopWords(words []string) {
	v.mu.Lock()
	defer v.mu.Unlock()
	for _, word := range words {
		v.stopWords[strings.ToLower(word)] = true
	}
}

// GetVocabularySize returns the current vocabulary size
func (v *TFIDFVectorizer) GetVocabularySize() int {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return len(v.vocabulary)
}

// IsFitted returns whether the vectorizer has been fitted
func (v *TFIDFVectorizer) IsFitted() bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.fitted
}
