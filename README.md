<p align="center"><img src="https://raw.githubusercontent.com/go-ruby-bleve/brand/main/social/go-ruby-bleve-bleve.png" alt="go-ruby-bleve/bleve" width="720"></p>

# bleve — go-ruby-bleve

[![Docs](https://img.shields.io/badge/docs-mkdocs--material-DC2626)](https://go-ruby-bleve.github.io/docs/)
[![License](https://img.shields.io/badge/license-BSD--3--Clause-blue)](LICENSE)
[![Go](https://img.shields.io/badge/go-1.26.4%2B-00ADD8)](https://go.dev/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100%25-1a7f37)](#tests--coverage)

**A pure-Go (no cgo) full-text search library** with a clean, idiomatic Ruby
search API. It indexes Hash-shaped documents in memory or on disk, and answers
match / phrase / term / prefix / fuzzy / wildcard / regexp / range / boolean /
query-string queries with scoring, highlighting and faceting — **without any C
extension and without a running search server**.

Unlike most `go-ruby-*` siblings, there is **no single canonical Ruby `bleve`
gem** to mirror: bleve is Go-native. This module therefore exposes that Go
capability through the shape a `Bleve` Ruby gem *would* have — `Bleve::Index`,
a `Bleve::Query` DSL, `Bleve::Mapping`, and Hash-shaped results — rather than
porting an existing library.

It builds on [`github.com/blevesearch/bleve/v2`](https://github.com/blevesearch/bleve)
(all indexing and searching is delegated to bleve) and is the search backend for
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby), but is a
**standalone, reusable** module with no dependency on the Ruby runtime.

> **CGO-free on every arch.** The default in-memory (gtreap) and on-disk (scorch
> + bbolt) indexes build and pass the full suite with `CGO_ENABLED=0` on all six
> supported 64-bit targets — amd64, arm64, riscv64, loong64, ppc64le and the
> big-endian **s390x**. bleve's optional faiss-backed vector search (which needs
> cgo) is gated behind a build tag and is not used.

## Features

- **In-memory and on-disk indexes** — `NewMemIndex` for ephemeral search,
  `New` / `Open` for a persistent index that survives close/reopen.
- **Hash documents** — index, fetch, and delete documents that are plain
  `map[string]interface{}`; batch many operations atomically.
- **Full query DSL** — `Match`, `MatchPhrase`, `Term`, `Prefix`, `Fuzzy`,
  `Wildcard`, `Regexp`, `NumericRange`, `DateRange`, `Bool`, `QueryString`,
  `MatchAll`, `MatchNone`, each with fluent `Field` / `Boost` / `Fuzziness`.
- **Rich results** — hits with id, score and stored fields; `Total`, `MaxScore`,
  `Took`; pagination (`Size` / `From`); sorting (`SortBy`); field selection.
- **Highlighting** — per-field match fragments via `Highlight` / `HighlightStyle`.
- **Faceting** — term, numeric-range and date-range aggregations via `WithFacet`.
- **Typed mappings** — text (with analyzers), keyword, numeric, datetime and
  boolean fields, or a zero-config dynamic default.

## Install

```sh
go get github.com/go-ruby-bleve/bleve
```

## Usage

```go
package main

import (
	"fmt"

	bleve "github.com/go-ruby-bleve/bleve"
)

func main() {
	// A typed mapping (or pass nil for the dynamic default).
	m := bleve.NewMapping().
		AddTextField("title").
		AddKeywordField("category").
		AddNumericField("views")

	idx, _ := bleve.NewMemIndex(m)
	defer idx.Close()

	idx.Index("post-1", map[string]interface{}{
		"title": "hello world", "category": "news", "views": 42.0,
	})
	idx.Index("post-2", map[string]interface{}{
		"title": "goodbye world", "category": "blog", "views": 7.0,
	})

	res, _ := idx.Search(
		bleve.Match("world").Field("title"),
		bleve.Size(10),
		bleve.Highlight(),
		bleve.WithFacet("by_category", bleve.TermFacet("category", 10)),
	)

	fmt.Println("total:", res.Total())
	for _, h := range res.Hits() {
		fmt.Printf("%s  score=%.3f  %v\n", h.ID, h.Score, h.Fragments["title"])
	}
	for _, tc := range res.Facets()["by_category"].Terms {
		fmt.Printf("category %q: %d\n", tc.Term, tc.Count)
	}
}
```

### Ruby shape (via rbgo)

```ruby
index = Bleve.new_mem_index(Bleve::Mapping.new.add_text_field("title"))
index.index("post-1", { "title" => "hello world", "views" => 42 })

res = index.search(Bleve::Query.match("world").field("title"),
                   size: 10, highlight: true)
res.hits.each { |h| puts "#{h.id} #{h.score}" }
```

## Query builders

| Builder | Purpose |
| --- | --- |
| `Match` / `MatchPhrase` | analyzed full-text match / ordered phrase |
| `Term` / `Prefix` | exact term / prefix match |
| `Fuzzy` / `Wildcard` / `Regexp` | edit-distance / glob / regex match |
| `NumericRange` / `DateRange` | half-open range over numeric / datetime fields |
| `Bool` | `must` (AND) / `should` (OR) / `must_not` (NOT) composition |
| `QueryString` | bleve query-string mini-language |
| `MatchAll` / `MatchNone` | match everything / nothing |

## Tests & coverage

The suite is deterministic, network-free (fixed UTC times, on-disk indexes under
`t.TempDir`) and holds **100 % statement coverage**, run with `-race`:

```sh
go test -race -cover ./...
```

CI runs nine jobs: `-race` + 100 %-coverage on Linux, macOS and Windows, plus the
six 64-bit targets (amd64/arm64 native; riscv64/loong64/ppc64le/s390x under
qemu-user) built with `CGO_ENABLED=0`.

## License

BSD-3-Clause — see [LICENSE](LICENSE). Copyright (c) 2026, the
go-ruby-bleve/bleve authors.
