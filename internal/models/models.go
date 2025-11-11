package models

import "time"

type Snapshot struct {
	Timestamp time.Time                  `json:"timestamp"`
	Env       string                     `json:"env"`
	Databases map[string]DatabaseSnapshot`json:"databases"`
}

type DatabaseSnapshot struct {
	DBName  string                    `json:"db_name"`
	Schemas map[string]SchemaSnapshot `json:"schemas"`
}

type SchemaSnapshot struct {
	Name   string                   `json:"name"`
	Tables map[string]TableSnapshot `json:"tables"`
}

type TableSnapshot struct {
	Name    string            `json:"name"`
	Columns map[string]string `json:"columns"` // col_name: data_type

	// NEW
	PrimaryKey       []string                     `json:"primary_key,omitempty"` // ordered PK columns
	UniqueConstraints map[string][]string         `json:"unique_constraints,omitempty"` // constraint_name -> ordered columns
	CheckConstraints  map[string]string           `json:"check_constraints,omitempty"`  // constraint_name -> definition
	ForeignKeys       map[string]ForeignKey       `json:"foreign_keys,omitempty"`       // fk_name -> fk
	Indexes           map[string]Index            `json:"indexes,omitempty"`            // index_name -> index
}

type ForeignKey struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns"`     // local columns (ordered)
	RefSchema  string   `json:"ref_schema"`
	RefTable   string   `json:"ref_table"`
	RefColumns []string `json:"ref_columns"` // referenced columns (ordered)
	UpdateRule string   `json:"update_rule,omitempty"` // NO ACTION / CASCADE / SET NULL / SET DEFAULT / RESTRICT
	DeleteRule string   `json:"delete_rule,omitempty"`
}

type Index struct {
	Name       string   `json:"name"`
	Columns    []string `json:"columns,omitempty"` // best-effort parsed
	Unique     bool     `json:"unique"`
	Definition string   `json:"definition"`        // full indexdef text
}
