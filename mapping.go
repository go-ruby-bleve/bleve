package bleve

import (
	blv "github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/mapping"
)

// Mapping describes how documents and their fields are indexed. It mirrors the
// Ruby Bleve::Mapping surface: start from a sensible default and, optionally,
// declare typed fields (text, keyword, numeric, datetime, boolean) and
// analyzers. A zero-configuration Mapping dynamically indexes every field.
type Mapping struct {
	impl *mapping.IndexMappingImpl
}

// NewMapping returns a Mapping seeded with bleve's default dynamic mapping.
func NewMapping() *Mapping {
	return &Mapping{impl: blv.NewIndexMapping()}
}

// SetDefaultAnalyzer sets the analyzer used for fields that do not name their
// own. Returns the Mapping for chaining.
func (m *Mapping) SetDefaultAnalyzer(name string) *Mapping {
	m.impl.DefaultAnalyzer = name
	return m
}

// AddTextField declares a tokenized, analyzed, stored text field.
func (m *Mapping) AddTextField(name string) *Mapping {
	m.impl.DefaultMapping.AddFieldMappingsAt(name, blv.NewTextFieldMapping())
	return m
}

// AddTextFieldWithAnalyzer declares a text field indexed with a named analyzer.
func (m *Mapping) AddTextFieldWithAnalyzer(name, analyzer string) *Mapping {
	fm := blv.NewTextFieldMapping()
	fm.Analyzer = analyzer
	m.impl.DefaultMapping.AddFieldMappingsAt(name, fm)
	return m
}

// AddKeywordField declares a non-tokenized text field (exact-match keyword).
func (m *Mapping) AddKeywordField(name string) *Mapping {
	m.impl.DefaultMapping.AddFieldMappingsAt(name, blv.NewKeywordFieldMapping())
	return m
}

// AddNumericField declares a numeric field (float64/int).
func (m *Mapping) AddNumericField(name string) *Mapping {
	m.impl.DefaultMapping.AddFieldMappingsAt(name, blv.NewNumericFieldMapping())
	return m
}

// AddDateTimeField declares a datetime field.
func (m *Mapping) AddDateTimeField(name string) *Mapping {
	m.impl.DefaultMapping.AddFieldMappingsAt(name, blv.NewDateTimeFieldMapping())
	return m
}

// AddBooleanField declares a boolean field.
func (m *Mapping) AddBooleanField(name string) *Mapping {
	m.impl.DefaultMapping.AddFieldMappingsAt(name, blv.NewBooleanFieldMapping())
	return m
}
