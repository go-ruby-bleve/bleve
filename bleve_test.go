package bleve

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	blv "github.com/blevesearch/bleve/v2"
)

// errFake is the sentinel returned by failIndex to exercise error paths.
var errFake = errors.New("boom")

// failIndex is a searchIndex whose operations all fail, used to drive the
// impl-error branches of Index deterministically. NewBatch delegates to a real
// index so a valid *blv.Batch can be produced before Batch itself fails.
type failIndex struct {
	real blv.Index
}

func (f failIndex) Index(string, interface{}) error                      { return errFake }
func (f failIndex) Delete(string) error                                  { return errFake }
func (f failIndex) DocCount() (uint64, error)                            { return 0, errFake }
func (f failIndex) NewBatch() *blv.Batch                                 { return f.real.NewBatch() }
func (f failIndex) Batch(*blv.Batch) error                               { return errFake }
func (f failIndex) Search(*blv.SearchRequest) (*blv.SearchResult, error) { return nil, errFake }
func (f failIndex) Close() error                                         { return errFake }

// newFailIndex returns an *Index whose impl always errors.
func newFailIndex(t *testing.T) *Index {
	t.Helper()
	real, err := blv.NewMemOnly(blv.NewIndexMapping())
	if err != nil {
		t.Fatalf("real mem index: %v", err)
	}
	t.Cleanup(func() { _ = real.Close() })
	return &Index{impl: failIndex{real: real}}
}

// sampleMapping declares the typed fields used by the corpus.
func sampleMapping() *Mapping {
	return NewMapping().
		AddTextField("title").
		AddTextFieldWithAnalyzer("body", "standard").
		AddKeywordField("category").
		AddNumericField("views").
		AddDateTimeField("published").
		AddBooleanField("featured").
		SetDefaultAnalyzer("standard")
}

func day(d int) time.Time {
	return time.Date(2020, time.January, d, 0, 0, 0, 0, time.UTC)
}

// corpus indexes a small deterministic document set into ix.
func loadCorpus(t *testing.T, ix *Index) {
	t.Helper()
	docs := []struct {
		id  string
		doc map[string]interface{}
	}{
		{"1", map[string]interface{}{"title": "hello world", "body": "the quick brown fox", "category": "news", "views": 10.0, "published": day(1), "featured": true}},
		{"2", map[string]interface{}{"title": "hello there", "body": "a lazy dog sleeps", "category": "news", "views": 50.0, "published": day(5), "featured": false}},
		{"3", map[string]interface{}{"title": "goodbye world", "body": "the quick red fox", "category": "blog", "views": 200.0, "published": day(10), "featured": true}},
		{"4", map[string]interface{}{"title": "unrelated topic", "body": "nothing to see", "category": "blog", "views": 999.0, "published": day(20), "featured": false}},
	}
	if err := ix.Batch(func(b *Batch) error {
		for _, d := range docs {
			if err := b.Index(d.id, d.doc); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("batch load: %v", err)
	}
}

func ids(res *SearchResult) map[string]bool {
	m := make(map[string]bool)
	for _, h := range res.Hits() {
		m[h.ID] = true
	}
	return m
}

func mustSearch(t *testing.T, ix *Index, q Query, opts ...SearchOption) *SearchResult {
	t.Helper()
	res, err := ix.Search(q, opts...)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	return res
}

// --- errors.go ---

func TestErrorAndWrap(t *testing.T) {
	e := &Error{Op: "index", Err: ErrClosed}
	if e.Error() != "bleve: index: bleve: index is closed" {
		t.Fatalf("Error() = %q", e.Error())
	}
	if !errors.Is(e, ErrClosed) {
		t.Fatal("errors.Is(ErrClosed) failed")
	}
	bare := &Error{Err: errFake}
	if bare.Error() != "boom" {
		t.Fatalf("bare Error() = %q", bare.Error())
	}
	if wrap("x", nil) != nil {
		t.Fatal("wrap(nil) must be nil")
	}
	if got := wrap("x", errFake); got == nil || !errors.Is(got, errFake) {
		t.Fatalf("wrap(non-nil) = %v", got)
	}
}

// --- mapping.go + construction ---

func TestNewMemIndexNilAndTypedMapping(t *testing.T) {
	ix, err := NewMemIndex(nil)
	if err != nil {
		t.Fatalf("nil mapping: %v", err)
	}
	if ix.Path() != "" {
		t.Fatalf("mem Path = %q", ix.Path())
	}
	if err := ix.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	ix2, err := NewMemIndex(sampleMapping())
	if err != nil {
		t.Fatalf("typed mapping: %v", err)
	}
	defer ix2.Close()
	loadCorpus(t, ix2)
	n, err := ix2.Count()
	if err != nil || n != 4 {
		t.Fatalf("count = %d, %v", n, err)
	}
}

func TestNewMemIndexError(t *testing.T) {
	_, err := NewMemIndex(NewMapping().SetDefaultAnalyzer("no-such-analyzer"))
	if err == nil {
		t.Fatal("expected analyzer error")
	}
}

// --- query builders + fluent setters ---

func TestQueryBuildersMatch(t *testing.T) {
	ix, _ := NewMemIndex(sampleMapping())
	defer ix.Close()
	loadCorpus(t, ix)

	// match on _all default field
	res := mustSearch(t, ix, Match("hello"))
	if res.Total() != 2 {
		t.Fatalf("match hello total = %d", res.Total())
	}
	// match with field + boost
	res = mustSearch(t, ix, Match("world").Field("title").Boost(2.0))
	if got := ids(res); !got["1"] || !got["3"] {
		t.Fatalf("match world/title ids = %v", got)
	}
	// match_phrase
	res = mustSearch(t, ix, MatchPhrase("hello world").Field("title"))
	if res.Total() != 1 || res.Hits()[0].ID != "1" {
		t.Fatalf("phrase total=%d", res.Total())
	}
	// term (keyword exact)
	res = mustSearch(t, ix, Term("news").Field("category"))
	if res.Total() != 2 {
		t.Fatalf("term news total=%d", res.Total())
	}
	// prefix
	res = mustSearch(t, ix, Prefix("good").Field("title"))
	if res.Total() != 1 || res.Hits()[0].ID != "3" {
		t.Fatalf("prefix good total=%d", res.Total())
	}
}

func TestQueryBuildersFuzzyWildcardRegexp(t *testing.T) {
	ix, _ := NewMemIndex(sampleMapping())
	defer ix.Close()
	loadCorpus(t, ix)

	// fuzzy: "helo" ~ "hello"
	res := mustSearch(t, ix, Fuzzy("helo").Field("title").Fuzziness(1))
	if res.Total() != 2 {
		t.Fatalf("fuzzy total=%d", res.Total())
	}
	// wildcard
	res = mustSearch(t, ix, Wildcard("wor*").Field("title"))
	if res.Total() != 2 {
		t.Fatalf("wildcard total=%d", res.Total())
	}
	// regexp
	res = mustSearch(t, ix, Regexp("hel.o").Field("title"))
	if res.Total() != 2 {
		t.Fatalf("regexp total=%d", res.Total())
	}
	// Fuzziness no-op branch on a term query (not fuzzy-capable)
	_ = Term("x").Fuzziness(2)
	// Field no-op branch on match_all (not fieldable)
	_ = MatchAll().Field("title")
}

func TestQueryBuildersRangeBoolStringAllNone(t *testing.T) {
	ix, _ := NewMemIndex(sampleMapping())
	defer ix.Close()
	loadCorpus(t, ix)

	// numeric range [10,100)
	res := mustSearch(t, ix, NumericRange(F64(10), F64(100)).Field("views"))
	if got := ids(res); !got["1"] || !got["2"] || got["3"] || got["4"] {
		t.Fatalf("numeric range ids=%v", got)
	}
	// date range
	res = mustSearch(t, ix, DateRange(day(1), day(6)).Field("published"))
	if got := ids(res); !got["1"] || !got["2"] || got["3"] {
		t.Fatalf("date range ids=%v", got)
	}
	// bool: must hello(title) should featured, must_not category=blog
	res = mustSearch(t, ix, Bool(
		[]Query{Match("hello").Field("title")},
		[]Query{Term("news").Field("category")},
		[]Query{Term("blog").Field("category")},
	))
	if got := ids(res); !got["1"] || !got["2"] {
		t.Fatalf("bool ids=%v", got)
	}
	// query_string
	res = mustSearch(t, ix, QueryString("title:world"))
	if res.Total() != 2 {
		t.Fatalf("query_string total=%d", res.Total())
	}
	// match_all / match_none
	if mustSearch(t, ix, MatchAll()).Total() != 4 {
		t.Fatal("match_all != 4")
	}
	if mustSearch(t, ix, MatchNone()).Total() != 0 {
		t.Fatal("match_none != 0")
	}
}

// --- search options, hits, highlight, facets ---

func TestSearchOptionsAndResultAccessors(t *testing.T) {
	ix, _ := NewMemIndex(sampleMapping())
	defer ix.Close()
	loadCorpus(t, ix)

	res := mustSearch(t, ix, MatchAll(),
		Size(2), From(0), SortBy("-views"), Fields("*"))
	if len(res.Hits()) != 2 {
		t.Fatalf("size 2 -> %d hits", len(res.Hits()))
	}
	if res.Hits()[0].ID != "4" {
		t.Fatalf("sort -views first id = %s", res.Hits()[0].ID)
	}
	if res.MaxScore() < 0 {
		t.Fatalf("max score = %f", res.MaxScore())
	}
	if res.Took() < 0 {
		t.Fatalf("took = %v", res.Took())
	}
	// paging From
	res2 := mustSearch(t, ix, MatchAll(), Size(2), From(2), SortBy("-views"))
	if res2.Hits()[0].ID == res.Hits()[0].ID {
		t.Fatal("From paging did not advance")
	}
	// hit fields populated
	if _, ok := res.Hits()[0].Fields["views"]; !ok {
		t.Fatalf("fields not loaded: %v", res.Hits()[0].Fields)
	}
}

func TestHighlight(t *testing.T) {
	ix, _ := NewMemIndex(sampleMapping())
	defer ix.Close()
	loadCorpus(t, ix)

	res := mustSearch(t, ix, Match("quick").Field("body"), Highlight(), Fields("*"))
	found := false
	for _, h := range res.Hits() {
		if frs, ok := h.Fragments["body"]; ok && len(frs) > 0 {
			found = true
		}
	}
	if !found {
		t.Fatal("no highlight fragments produced")
	}
	// styled highlight also works
	res2 := mustSearch(t, ix, Match("quick").Field("body"), HighlightStyle("html"))
	if res2.Total() != 2 {
		t.Fatalf("styled highlight total=%d", res2.Total())
	}
}

func TestFacets(t *testing.T) {
	ix, _ := NewMemIndex(sampleMapping())
	defer ix.Close()
	loadCorpus(t, ix)

	res := mustSearch(t, ix, MatchAll(),
		WithFacet("by_category", TermFacet("category", 5)),
		WithFacet("by_views", TermFacet("views", 5).
			AddNumericRange("low", F64(0), F64(100)).
			AddNumericRange("high", F64(100), nil)),
		WithFacet("by_date", TermFacet("published", 5).
			AddDateRange("early", day(1), day(8)).
			AddDateRange("late", day(8), day(31))),
	)
	f := res.Facets()
	cat := f["by_category"]
	if cat == nil || cat.Field != "category" {
		t.Fatalf("category facet = %+v", cat)
	}
	counts := map[string]int{}
	for _, tc := range cat.Terms {
		counts[tc.Term] = tc.Count
	}
	if counts["news"] != 2 || counts["blog"] != 2 {
		t.Fatalf("term facet counts = %v", counts)
	}
	if cat.Total == 0 {
		t.Fatalf("facet total=%d", cat.Total)
	}
	views := f["by_views"]
	if len(views.NumericRanges) != 2 {
		t.Fatalf("numeric ranges = %+v", views.NumericRanges)
	}
	dates := f["by_date"]
	if len(dates.DateRanges) != 2 {
		t.Fatalf("date ranges = %+v", dates.DateRanges)
	}
	_ = views.Missing
	_ = dates.Other
}

// --- document, delete, count ---

func TestDocumentDeleteCount(t *testing.T) {
	ix, _ := NewMemIndex(sampleMapping())
	defer ix.Close()
	loadCorpus(t, ix)

	doc, err := ix.Document("1")
	if err != nil {
		t.Fatalf("document: %v", err)
	}
	if doc["title"] == nil {
		t.Fatalf("document fields = %v", doc)
	}
	// not found
	if _, err := ix.Document("nope"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	// single index + delete
	if err := ix.Index("5", map[string]interface{}{"title": "temp doc"}); err != nil {
		t.Fatalf("index: %v", err)
	}
	n, _ := ix.Count()
	if n != 5 {
		t.Fatalf("count after add = %d", n)
	}
	if err := ix.Delete("5"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	res := mustSearch(t, ix, Match("temp").Field("title"))
	if res.Total() != 0 {
		t.Fatalf("deleted doc still found: %d", res.Total())
	}
}

// --- on-disk persistence ---

func TestOnDiskPersistence(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idx.bleve")

	ix, err := New(path, sampleMapping())
	if err != nil {
		t.Fatalf("new on-disk: %v", err)
	}
	if ix.Path() != path {
		t.Fatalf("path = %q", ix.Path())
	}
	loadCorpus(t, ix)
	if err := ix.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	// New on an existing path must fail.
	if _, err := New(path, sampleMapping()); err == nil {
		t.Fatal("New on existing path should error")
	}

	reopened, err := Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer reopened.Close()
	n, err := reopened.Count()
	if err != nil || n != 4 {
		t.Fatalf("persisted count = %d, %v", n, err)
	}
	res := mustSearch(t, reopened, Match("hello").Field("title"))
	if res.Total() != 2 {
		t.Fatalf("persisted search total=%d", res.Total())
	}
}

func TestNewOnDiskNilMapping(t *testing.T) {
	path := filepath.Join(t.TempDir(), "idx.bleve")
	ix, err := New(path, nil)
	if err != nil {
		t.Fatalf("new nil mapping: %v", err)
	}
	if err := ix.Index("a", map[string]interface{}{"title": "hi"}); err != nil {
		t.Fatalf("index: %v", err)
	}
	if err := ix.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
}

func TestOpenError(t *testing.T) {
	if _, err := Open(filepath.Join(t.TempDir(), "does-not-exist")); err == nil {
		t.Fatal("Open of missing path should error")
	}
}

// --- closed-index guards ---

func TestClosedGuards(t *testing.T) {
	ix, _ := NewMemIndex(nil)
	if err := ix.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := ix.Close(); !errors.Is(err, ErrClosed) {
		t.Fatalf("double close: %v", err)
	}
	if err := ix.Index("a", nil); !errors.Is(err, ErrClosed) {
		t.Fatalf("index closed: %v", err)
	}
	if err := ix.Delete("a"); !errors.Is(err, ErrClosed) {
		t.Fatalf("delete closed: %v", err)
	}
	if _, err := ix.Count(); !errors.Is(err, ErrClosed) {
		t.Fatalf("count closed: %v", err)
	}
	if _, err := ix.Document("a"); !errors.Is(err, ErrClosed) {
		t.Fatalf("document closed: %v", err)
	}
	if _, err := ix.Search(MatchAll()); !errors.Is(err, ErrClosed) {
		t.Fatalf("search closed: %v", err)
	}
	if err := ix.Batch(func(*Batch) error { return nil }); !errors.Is(err, ErrClosed) {
		t.Fatalf("batch closed: %v", err)
	}
}

// --- impl-error guards via failIndex ---

func TestImplErrors(t *testing.T) {
	ix := newFailIndex(t)
	if err := ix.Index("a", nil); !errors.Is(err, errFake) {
		t.Fatalf("index err = %v", err)
	}
	if err := ix.Delete("a"); !errors.Is(err, errFake) {
		t.Fatalf("delete err = %v", err)
	}
	if _, err := ix.Count(); !errors.Is(err, errFake) {
		t.Fatalf("count err = %v", err)
	}
	if _, err := ix.Document("a"); !errors.Is(err, errFake) {
		t.Fatalf("document err = %v", err)
	}
	if _, err := ix.Search(MatchAll()); !errors.Is(err, errFake) {
		t.Fatalf("search err = %v", err)
	}
	if err := ix.Close(); !errors.Is(err, errFake) {
		t.Fatalf("close err = %v", err)
	}
}

func TestBatchErrors(t *testing.T) {
	// fn returns an error -> not applied
	ix, _ := NewMemIndex(nil)
	defer ix.Close()
	sentinel := errors.New("stop")
	if err := ix.Batch(func(b *Batch) error {
		_ = b.Index("x", map[string]interface{}{"t": "y"})
		b.Delete("z")
		return sentinel
	}); !errors.Is(err, sentinel) {
		t.Fatalf("batch fn error = %v", err)
	}
	if n, _ := ix.Count(); n != 0 {
		t.Fatalf("failed batch was applied, count=%d", n)
	}

	// impl.Batch fails
	fi := newFailIndex(t)
	if err := fi.Batch(func(b *Batch) error {
		return b.Index("x", map[string]interface{}{"t": "y"})
	}); !errors.Is(err, errFake) {
		t.Fatalf("batch impl error = %v", err)
	}
}
