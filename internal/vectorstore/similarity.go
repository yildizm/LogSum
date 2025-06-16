package vectorstore

import (
	"math"
)

// CosineSimilarity calculates the cosine similarity between two vectors
// Returns a value between -1 and 1, where 1 means identical vectors
func CosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0.0
	}

	var dotProduct, normA, normB float32

	for i := 0; i < len(a); i++ {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0.0 || normB == 0.0 {
		return 0.0
	}

	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

// EuclideanDistance calculates the Euclidean distance between two vectors
// Lower values indicate higher similarity
func EuclideanDistance(a, b []float32) float32 {
	if len(a) != len(b) {
		return float32(math.Inf(1))
	}

	var sum float32
	for i := 0; i < len(a); i++ {
		diff := a[i] - b[i]
		sum += diff * diff
	}

	return float32(math.Sqrt(float64(sum)))
}

// DotProduct calculates the dot product of two vectors
func DotProduct(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0.0
	}

	var product float32
	for i := 0; i < len(a); i++ {
		product += a[i] * b[i]
	}

	return product
}

// NormalizeVector normalizes a vector to unit length
func NormalizeVector(v []float32) []float32 {
	var norm float32
	for _, val := range v {
		norm += val * val
	}

	norm = float32(math.Sqrt(float64(norm)))
	if norm == 0.0 {
		return v
	}

	normalized := make([]float32, len(v))
	for i, val := range v {
		normalized[i] = val / norm
	}

	return normalized
}

// MagnitudeSquared calculates the squared magnitude of a vector
func MagnitudeSquared(v []float32) float32 {
	var sum float32
	for _, val := range v {
		sum += val * val
	}
	return sum
}

// Magnitude calculates the magnitude (length) of a vector
func Magnitude(v []float32) float32 {
	return float32(math.Sqrt(float64(MagnitudeSquared(v))))
}
