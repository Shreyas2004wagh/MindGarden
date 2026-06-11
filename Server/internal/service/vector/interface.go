package vector

type VectorStore interface {
	Add(doc Document) error
	Save() error
	Load() error
	Search(query Vector, k int) ([]Document, error)
}
