package generator

import (
	"bytes"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Saba101/GoMetaSync/internal/models"
)

var fileTmpl = template.Must(template.New("file").Parse(`// Code generated automatically. DO NOT EDIT.
package generated_models

{{- if .NeedsTime }}
import "time"
{{- end }}

type {{.StructName}} struct {
{{- range .Fields }}
    {{ .Name }} {{ .Type }} ` + "`json:\"{{ .JSONName }}\"`" + `
{{- end }}
}
`))

type Field struct {
	Name     string
	Type     string
	JSONName string
}

type TableTemplateData struct {
	StructName string
	Fields     []Field
	NeedsTime  bool
}

func GenerateStructs(snap *models.Snapshot, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	for dbName, db := range snap.Databases {
		for schemaName, schema := range db.Schemas {
			schemaDir := filepath.Join(outDir, sanitize(dbName), sanitize(schemaName))
			if err := os.MkdirAll(schemaDir, 0o755); err != nil {
				return err
			}

			for tableName, table := range schema.Tables {
				// stable order of columns
				colNames := make([]string, 0, len(table.Columns))
				for c := range table.Columns {
					colNames = append(colNames, c)
				}
				sort.Strings(colNames)

				fields := make([]Field, 0, len(colNames))
				needsTime := false

				for _, col := range colNames {
					goType := mapSQLType(table.Columns[col])
					if goType == "time.Time" {
						needsTime = true
					}
					fields = append(fields, Field{
						Name:     toPascalCase(col),
						Type:     goType,
						JSONName: col,
					})
				}

				data := TableTemplateData{
					StructName: toPascalCase(tableName),
					Fields:     fields,
					NeedsTime:  needsTime,
				}

				// render
				var buf bytes.Buffer
				if err := fileTmpl.Execute(&buf, data); err != nil {
					return err
				}

				// write
				filePath := filepath.Join(schemaDir, toPascalCase(tableName)+".go")
				if err := os.WriteFile(filePath, buf.Bytes(), 0o644); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// ---- helpers ----

func sanitize(s string) string {
	// drop characters that make bad folder names
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.ReplaceAll(s, ".", "_")
	s = strings.ReplaceAll(s, "-", "_")
	return s
}

// Convert snake_case to PascalCase
func toPascalCase(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-' || r == ' '
	})
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	return strings.Join(parts, "")
}

// Basic SQL → Go type mapper
func mapSQLType(sqlType string) string {
	t := strings.ToLower(strings.TrimSpace(sqlType))
	// normalize common postgres types
	switch {
	case strings.Contains(t, "int"):
		return "int"
	case strings.Contains(t, "bigint"):
		return "int64"
	case strings.Contains(t, "smallint"):
		return "int16"
	case strings.Contains(t, "double"), strings.Contains(t, "real"), strings.Contains(t, "float"):
		return "float64"
	case strings.Contains(t, "numeric"), strings.Contains(t, "decimal"):
		// you can later switch this to github.com/shopspring/decimal
		return "float64"
	case strings.Contains(t, "bool"):
		return "bool"
	case strings.Contains(t, "timestamp"), strings.Contains(t, "date"), strings.Contains(t, "time"):
		return "time.Time"
	case strings.Contains(t, "json"):
		return "map[string]interface{}"
	case strings.Contains(t, "uuid"):
		return "string"
	case strings.Contains(t, "char"), strings.Contains(t, "text"), strings.Contains(t, "enum"):
		return "string"
	case strings.Contains(t, "bytea"):
		return "[]byte"
	default:
		return "interface{}"
	}
}
