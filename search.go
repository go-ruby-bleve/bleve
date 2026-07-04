package bleve

import (
	"time"

	blv "github.com/blevesearch/bleve/v2"
)

// searchOpts accumulates the effect of the functional SearchOption values.
type searchOpts struct {
	size      int
	from      int
	fields    []string
	highlight *blv.HighlightRequest
	sort      []string
	facets    map[string]*blv.FacetRequest
}

// SearchOption configures a Search call. Mirrors the Ruby keyword arguments
// size:, from:, fields:, highlight:, sort:, facets:.
type SearchOption func(*searchOpts)

// Size sets the maximum number of hits to return.
func Size(n int) SearchOption { return func(o *searchOpts) { o.size = n } }

// From sets the offset of the first hit to return (for pagination).
func From(n int) SearchOption { return func(o *searchOpts) { o.from = n } }

// Fields requests that the given stored fields be loaded onto each hit. Use
// "*" to load every stored field.
func Fields(fields ...string) SearchOption {
	return func(o *searchOpts) { o.fields = fields }
}

// Highlight enables highlighting with the default fragmenter/formatter.
func Highlight() SearchOption {
	return func(o *searchOpts) { o.highlight = blv.NewHighlight() }
}

// HighlightStyle enables highlighting with a named style (e.g. "html", "ansi").
func HighlightStyle(style string) SearchOption {
	return func(o *searchOpts) { o.highlight = blv.NewHighlightWithStyle(style) }
}

// SortBy sets the sort order. Each entry is a field name; prefix with "-" for
// descending, and use "_score" / "_id" for the built-in sorts.
func SortBy(fields ...string) SearchOption {
	return func(o *searchOpts) { o.sort = fields }
}

// WithFacet adds a named facet (aggregation) to the search.
func WithFacet(name string, f *Facet) SearchOption {
	return func(o *searchOpts) {
		if o.facets == nil {
			o.facets = make(map[string]*blv.FacetRequest)
		}
		o.facets[name] = f.inner
	}
}

// Facet describes an aggregation over the result set. Build a term facet with
// TermFacet, then optionally attach numeric or date buckets.
type Facet struct {
	inner *blv.FacetRequest
}

// TermFacet counts the top-size distinct values of field.
func TermFacet(field string, size int) *Facet {
	return &Facet{inner: blv.NewFacetRequest(field, size)}
}

// AddNumericRange adds a named numeric bucket [min, max) to the facet. A nil
// bound is unbounded.
func (f *Facet) AddNumericRange(name string, min, max *float64) *Facet {
	f.inner.AddNumericRange(name, min, max)
	return f
}

// AddDateRange adds a named date bucket [start, end) to the facet.
func (f *Facet) AddDateRange(name string, start, end time.Time) *Facet {
	f.inner.AddDateTimeRange(name, start, end)
	return f
}

// Search runs q against the index with the given options and returns a
// SearchResult (Bleve::Index#search).
func (ix *Index) Search(q Query, opts ...SearchOption) (*SearchResult, error) {
	if ix.closed {
		return nil, &Error{Op: "search", Err: ErrClosed}
	}
	o := &searchOpts{}
	for _, opt := range opts {
		opt(o)
	}
	req := blv.NewSearchRequest(q.inner)
	if o.size > 0 {
		req.Size = o.size
	}
	req.From = o.from
	if len(o.fields) > 0 {
		req.Fields = o.fields
	}
	if o.highlight != nil {
		req.Highlight = o.highlight
	}
	if len(o.sort) > 0 {
		req.SortBy(o.sort)
	}
	for name, fr := range o.facets {
		req.AddFacet(name, fr)
	}
	res, err := ix.impl.Search(req)
	if err != nil {
		return nil, wrap("search", err)
	}
	return &SearchResult{raw: res}, nil
}

// Hit is a single search result: the document id, its relevance score, any
// requested stored fields (as a Hash) and, when highlighting is on, the
// highlighted fragments per field.
type Hit struct {
	ID        string
	Score     float64
	Fields    map[string]interface{}
	Fragments map[string][]string
}

// SearchResult is the outcome of a Search (Bleve::SearchResult).
type SearchResult struct {
	raw *blv.SearchResult
}

// Total is the number of documents that matched (independent of paging).
func (r *SearchResult) Total() uint64 { return r.raw.Total }

// MaxScore is the highest score among the matches.
func (r *SearchResult) MaxScore() float64 { return r.raw.MaxScore }

// Took is the wall-clock time the search took.
func (r *SearchResult) Took() time.Duration { return r.raw.Took }

// Hits returns the page of matches.
func (r *SearchResult) Hits() []Hit {
	hits := make([]Hit, len(r.raw.Hits))
	for i, h := range r.raw.Hits {
		hits[i] = Hit{
			ID:        h.ID,
			Score:     h.Score,
			Fields:    h.Fields,
			Fragments: map[string][]string(h.Fragments),
		}
	}
	return hits
}

// TermCount is one bucket of a term facet.
type TermCount struct {
	Term  string
	Count int
}

// RangeCount is one bucket of a numeric or date range facet.
type RangeCount struct {
	Name  string
	Count int
}

// FacetResult is the aggregated result for a single named facet.
type FacetResult struct {
	Field         string
	Total         int
	Missing       int
	Other         int
	Terms         []TermCount
	NumericRanges []RangeCount
	DateRanges    []RangeCount
}

// Facets returns the computed facets keyed by the name given to WithFacet.
func (r *SearchResult) Facets() map[string]*FacetResult {
	out := make(map[string]*FacetResult, len(r.raw.Facets))
	for name, f := range r.raw.Facets {
		fr := &FacetResult{
			Field:   f.Field,
			Total:   f.Total,
			Missing: f.Missing,
			Other:   f.Other,
		}
		if f.Terms != nil {
			for _, t := range f.Terms.Terms() {
				fr.Terms = append(fr.Terms, TermCount{Term: t.Term, Count: t.Count})
			}
		}
		for _, nr := range f.NumericRanges {
			fr.NumericRanges = append(fr.NumericRanges, RangeCount{Name: nr.Name, Count: nr.Count})
		}
		for _, dr := range f.DateRanges {
			fr.DateRanges = append(fr.DateRanges, RangeCount{Name: dr.Name, Count: dr.Count})
		}
		out[name] = fr
	}
	return out
}
