// Package bleve is a pure-Go (CGO=0) full-text search library for the
// Ruby / rbgo ecosystem.
//
// # Not a port of an existing gem
//
// Unlike most go-ruby-* siblings, there is no single canonical Ruby "bleve"
// gem to mirror: bleve is a Go-native search engine. This package therefore
// exposes that Go capability through a clean, idiomatic Ruby-shaped search API
// -- the shape a hypothetical Bleve Ruby gem would have -- rather than
// reproducing a specific upstream Ruby library.
//
// It is a thin, ergonomic wrapper over github.com/blevesearch/bleve/v2. All
// indexing and searching is delegated to bleve; this package only provides the
// Hash-shaped documents/results, snake_case-friendly method names, and query
// DSL a Ruby developer would expect. When bound into rbgo it surfaces as:
//
//	index = Bleve.new_mem_index(Bleve::Mapping.new)
//	index.index("doc-1", { "title" => "hello world", "views" => 42 })
//	res = index.search(Bleve::Query.match("hello"), size: 10, highlight: true)
//	res.hits.each { |h| puts "#{h.id} #{h.score}" }
//
// # Go surface
//
// Index construction:
//
//	NewMemIndex(mapping)   // in-memory (Bleve.new_mem_index)
//	New(path, mapping)     // on-disk, fresh   (Bleve.new)
//	Open(path)             // on-disk, existing (Bleve.open)
//
// Index operations: Index, Batch, Delete, Document, Count, Close, Search.
//
// Query builders (Bleve::Query.*): Match, MatchPhrase, Term, Prefix, Fuzzy,
// Wildcard, Regexp, NumericRange, DateRange, Bool, QueryString, MatchAll,
// MatchNone. Each returns a Query with fluent Field/Boost/Fuzziness setters.
//
// Search options: Size, From, Fields, Highlight, HighlightStyle, SortBy,
// WithFacet. Results are a SearchResult exposing Hits, Total, MaxScore, Took
// and Facets.
//
// Mapping (Bleve::Mapping) configures document and field mappings (text,
// keyword, numeric, datetime, boolean) and analyzers, or a sensible default.
//
// # Portability
//
// The default in-memory (gtreap) and on-disk (scorch) indexes build with
// CGO_ENABLED=0 on every supported 64-bit target -- amd64, arm64, riscv64,
// loong64, ppc64le and s390x (big-endian) -- so the library is fully pure-Go.
// bleve's optional faiss-backed vector search (which needs cgo) is gated behind
// a build tag and is not used here.
package bleve
