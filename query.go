package bleve

import (
	"time"

	blv "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

// Query wraps a bleve query. Build one with the package-level constructors
// (mirroring Bleve::Query.*) and, where applicable, refine it with the fluent
// Field, Boost and Fuzziness setters.
type Query struct {
	inner query.Query
}

// F64 returns a pointer to v. It is a convenience for the open-ended bounds of
// NumericRange (a nil bound means unbounded on that side).
func F64(v float64) *float64 { return &v }

// Match builds an analyzed full-text match query (Bleve::Query.match).
func Match(text string) Query { return Query{blv.NewMatchQuery(text)} }

// MatchPhrase builds a phrase query preserving term order and adjacency.
func MatchPhrase(text string) Query { return Query{blv.NewMatchPhraseQuery(text)} }

// Term builds an exact, un-analyzed term query.
func Term(term string) Query { return Query{blv.NewTermQuery(term)} }

// Prefix builds a prefix query.
func Prefix(prefix string) Query { return Query{blv.NewPrefixQuery(prefix)} }

// Fuzzy builds a fuzzy (edit-distance) query.
func Fuzzy(term string) Query { return Query{blv.NewFuzzyQuery(term)} }

// Wildcard builds a wildcard query (* and ? metacharacters).
func Wildcard(pattern string) Query { return Query{blv.NewWildcardQuery(pattern)} }

// Regexp builds a regular-expression query.
func Regexp(pattern string) Query { return Query{blv.NewRegexpQuery(pattern)} }

// NumericRange builds a numeric range query. A nil bound is unbounded; use F64
// to take the address of a literal.
func NumericRange(min, max *float64) Query { return Query{blv.NewNumericRangeQuery(min, max)} }

// DateRange builds a date range query over [start, end).
func DateRange(start, end time.Time) Query { return Query{blv.NewDateRangeQuery(start, end)} }

// QueryString builds a query from bleve's query-string mini-language.
func QueryString(s string) Query { return Query{blv.NewQueryStringQuery(s)} }

// MatchAll matches every document.
func MatchAll() Query { return Query{blv.NewMatchAllQuery()} }

// MatchNone matches no document.
func MatchNone() Query { return Query{blv.NewMatchNoneQuery()} }

// Bool builds a boolean query from must (AND), should (OR) and mustNot (NOT)
// sub-queries. Any slice may be nil.
func Bool(must, should, mustNot []Query) Query {
	bq := blv.NewBooleanQuery()
	for _, q := range must {
		bq.AddMust(q.inner)
	}
	for _, q := range should {
		bq.AddShould(q.inner)
	}
	for _, q := range mustNot {
		bq.AddMustNot(q.inner)
	}
	return Query{bq}
}

// Field restricts the query to a single field. It is a no-op for queries that
// are not field-scoped (e.g. MatchAll, Bool, QueryString).
func (q Query) Field(field string) Query {
	if fq, ok := q.inner.(query.FieldableQuery); ok {
		fq.SetField(field)
	}
	return q
}

// Boost scales the query's contribution to the score.
func (q Query) Boost(b float64) Query {
	q.inner.(query.BoostableQuery).SetBoost(b)
	return q
}

// Fuzziness sets the allowed edit distance. It is a no-op for query types that
// do not support fuzziness.
func (q Query) Fuzziness(f int) Query {
	if fq, ok := q.inner.(interface{ SetFuzziness(int) }); ok {
		fq.SetFuzziness(f)
	}
	return q
}
