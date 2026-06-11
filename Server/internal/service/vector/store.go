package vector

import (
	"encoding/json"
	"math"
	"os"
	"sync"
)

type Vector []float32

type Document struct {
	ID        string                 `json:"id"`
	Embedding Vector                 `json:"embedding"`
	Metadata  map[string]interface{} `json:"metadata"`
	Content   string                 `json:"content"`
}

type MemoryStore struct {
	mu        sync.RWMutex
	Documents []Document `json:"documents"`
	Filepath  string
}

func NewMemoryStore(filepath string) *MemoryStore {
	return &MemoryStore{
		Documents: []Document{},
		Filepath:  filepath,
	}
}

func (s *MemoryStore) Add(doc Document) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Documents = append(s.Documents, doc)
	return nil
}

func (s *MemoryStore) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.Marshal(s.Documents)
	if err != nil {
		return err
	}
	return os.WriteFile(s.Filepath, data, 0644)
}

func (s *MemoryStore) Load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.Filepath)
	if os.IsNotExist(err) {
		s.Documents = []Document{}
		return nil
	}
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &s.Documents)
}

func CosineSimilarity(a, b Vector) float32 {
	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB))))
}

func (s *MemoryStore) Search(query Vector, k int) ([]Document, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type result struct {
		doc   Document
		score float32
	}

	results := make([]result, len(s.Documents))
	for i, doc := range s.Documents {
		results[i] = result{
			doc:   doc,
			score: CosineSimilarity(query, doc.Embedding),
		}
	}

	// Sort by score descending (simple bubble sort for small k/N, or use proper sort)
	// For MVP simplicity:
	for i := 0; i < len(results)-1; i++ {
		for j := 0; j < len(results)-i-1; j++ {
			if results[j].score < results[j+1].score {
				results[j], results[j+1] = results[j+1], results[j]
			}
		}
	}

	if k > len(results) {
		k = len(results)
	}

	topK := make([]Document, k)
	for i := 0; i < k; i++ {
		topK[i] = results[i].doc
	}

	return topK, nil
}
