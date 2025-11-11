package models

import "time"

type Snapshot struct {
    Timestamp time.Time                      `json:"timestamp"`
    Env       string                          `json:"env"`
    Databases map[string]DatabaseSnapshot     `json:"databases"`
}

type DatabaseSnapshot struct {
    DBName  string                        `json:"db_name"`
    Schemas map[string]SchemaSnapshot     `json:"schemas"`
}

type SchemaSnapshot struct {
    Name   string                    `json:"name"`
    Tables map[string]TableSnapshot  `json:"tables"`
}

type TableSnapshot struct {
    Name    string            `json:"name"`
    Columns map[string]string `json:"columns"` // col_name: data_type
}
