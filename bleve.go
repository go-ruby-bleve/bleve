package bleve

import (
	blv "github.com/blevesearch/bleve/v2"
)

// searchIndex is the subset of blv.Index this package drives. It exists as a
// seam so failure paths can be exercised deterministically in tests; the real
// bleve index satisfies it directly.
type searchIndex interface {
	Index(id string, data interface{}) error
	Delete(id string) error
	DocCount() (uint64, error)
	NewBatch() *blv.Batch
	Batch(b *blv.Batch) error
	Search(req *blv.SearchRequest) (*blv.SearchResult, error)
	Close() error
}

// Index is a full-text index. It mirrors the Ruby Bleve::Index surface: index
// Hash-shaped documents, search them, and (for on-disk indexes) persist across
// open/close. An Index is safe to use until Close is called.
type Index struct {
	impl   searchIndex
	path   string
	closed bool
}

// NewMemIndex creates an in-memory index (Bleve.new_mem_index). Pass nil to use
// the default dynamic mapping. Nothing is written to disk.
func NewMemIndex(m *Mapping) (*Index, error) {
	if m == nil {
		m = NewMapping()
	}
	idx, err := blv.NewMemOnly(m.impl)
	if err != nil {
		return nil, wrap("new_mem_index", err)
	}
	return &Index{impl: idx}, nil
}

// New creates a fresh on-disk index at path (Bleve.new). It is an error if path
// already exists. Pass nil for the default dynamic mapping.
func New(path string, m *Mapping) (*Index, error) {
	if m == nil {
		m = NewMapping()
	}
	idx, err := blv.New(path, m.impl)
	if err != nil {
		return nil, wrap("new", err)
	}
	return &Index{impl: idx, path: path}, nil
}

// Open opens an existing on-disk index at path (Bleve.open).
func Open(path string) (*Index, error) {
	idx, err := blv.Open(path)
	if err != nil {
		return nil, wrap("open", err)
	}
	return &Index{impl: idx, path: path}, nil
}

// Path returns the on-disk location of the index, or "" for a memory index.
func (ix *Index) Path() string { return ix.path }

// Index adds or replaces the document with the given id. The document is a Hash
// (map) of field name to value.
func (ix *Index) Index(id string, doc map[string]interface{}) error {
	if ix.closed {
		return &Error{Op: "index", Err: ErrClosed}
	}
	return wrap("index", ix.impl.Index(id, doc))
}

// Delete removes the document with the given id. Deleting an absent id is not
// an error.
func (ix *Index) Delete(id string) error {
	if ix.closed {
		return &Error{Op: "delete", Err: ErrClosed}
	}
	return wrap("delete", ix.impl.Delete(id))
}

// Count returns the number of documents in the index.
func (ix *Index) Count() (uint64, error) {
	if ix.closed {
		return 0, &Error{Op: "count", Err: ErrClosed}
	}
	n, err := ix.impl.DocCount()
	if err != nil {
		return 0, wrap("count", err)
	}
	return n, nil
}

// Document returns the stored fields of the document with the given id as a
// Hash. It returns an error wrapping ErrNotFound if the id is absent.
func (ix *Index) Document(id string) (map[string]interface{}, error) {
	if ix.closed {
		return nil, &Error{Op: "document", Err: ErrClosed}
	}
	req := blv.NewSearchRequest(blv.NewDocIDQuery([]string{id}))
	req.Fields = []string{"*"}
	res, err := ix.impl.Search(req)
	if err != nil {
		return nil, wrap("document", err)
	}
	if res.Total == 0 {
		return nil, &Error{Op: "document", Err: ErrNotFound}
	}
	return res.Hits[0].Fields, nil
}

// Close releases the index. Further operations return ErrClosed.
func (ix *Index) Close() error {
	if ix.closed {
		return &Error{Op: "close", Err: ErrClosed}
	}
	ix.closed = true
	return wrap("close", ix.impl.Close())
}

// Batch is an accumulator for a group of index/delete operations applied
// atomically. Obtain one via Index.Batch.
type Batch struct {
	inner *blv.Batch
}

// Index queues a document for indexing in the batch.
func (b *Batch) Index(id string, doc map[string]interface{}) error {
	return b.inner.Index(id, doc)
}

// Delete queues a document id for deletion in the batch.
func (b *Batch) Delete(id string) {
	b.inner.Delete(id)
}

// Batch runs fn against a Batch and, if fn returns nil, applies all queued
// operations atomically (Bleve::Index#batch { |b| ... }).
func (ix *Index) Batch(fn func(*Batch) error) error {
	if ix.closed {
		return &Error{Op: "batch", Err: ErrClosed}
	}
	b := &Batch{inner: ix.impl.NewBatch()}
	if err := fn(b); err != nil {
		return wrap("batch", err)
	}
	return wrap("batch", ix.impl.Batch(b.inner))
}
