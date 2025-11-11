package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/Saba101/GoMetaSync/internal/models"
)

// GenerateStructs writes Go structs into outDir, one file per table, now enriched with constraint/index metadata.
func GenerateStructs(snap *models.Snapshot, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	for dbName, db := range snap.Databases {
		for schemaName, schema := range db.Schemas {
			for tableName, table := range schema.Tables {
				filename := filepath.Join(outDir, fmt.Sprintf("%s_%s_%s.go",
					sanitize(dbName), sanitize(schemaName), sanitize(tableName)))

				src, err := renderTableFile(dbName, schemaName, &table)
				if err != nil {
					return err
				}
				if err := os.WriteFile(filename, []byte(src), 0o644); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func renderTableFile(dbName, schemaName string, t *models.TableSnapshot) (string, error) {
	// Build per-column metadata sets
	pkSet := make(map[string]bool, len(t.PrimaryKey))
	for _, c := range t.PrimaryKey {
		pkSet[c] = true
	}

	// unique constraints: column -> list of constraint names in column order (best-effort)
	colToUQ := map[string][]string{}
	for cname, cols := range t.UniqueConstraints {
		for _, c := range cols {
			colToUQ[c] = append(colToUQ[c], cname)
		}
	}
	// foreign keys: column -> list of "fkname:ref_schema.ref_table(ref_cols)"
	colToFK := map[string][]string{}
	for fkName, fk := range t.ForeignKeys {
		refSpec := fmt.Sprintf("%s.%s(%s)", fk.RefSchema, fk.RefTable, strings.Join(fk.RefColumns, ","))
		spec := fmt.Sprintf("%s:%s", fkName, refSpec)
		for _, c := range fk.Columns {
			colToFK[c] = append(colToFK[c], spec)
		}
	}
	// indexes: column -> list of index names (best-effort)
	colToIDX := map[string][]string{}
	for idxName, idx := range t.Indexes {
		for _, c := range idx.Columns {
			colToIDX[c] = append(colToIDX[c], idxName)
		}
	}

	// Sorted column order (stable output)
	colNames := make([]string, 0, len(t.Columns))
	for c := range t.Columns {
		colNames = append(colNames, c)
	}
	sort.Strings(colNames)

	// Build fields
	fields := make([]field, 0, len(colNames))
	for _, col := range colNames {
		goType := mapPgTypeToGo(t.Columns[col])
		tags := []string{
			fmt.Sprintf(`json:"%s"`, col),
			fmt.Sprintf(`db:"%s"`, col),
		}
		if pkSet[col] {
			tags = append(tags, `pk:"true"`)
		}
		if uq := colToUQ[col]; len(uq) > 0 {
			sort.Strings(uq)
			tags = append(tags, fmt.Sprintf(`unique:"%s"`, strings.Join(uq, ",")))
		}
		if fks := colToFK[col]; len(fks) > 0 {
			sort.Strings(fks)
			tags = append(tags, fmt.Sprintf(`fk:"%s"`, strings.Join(fks, ",")))
		}
		if idxs := colToIDX[col]; len(idxs) > 0 {
			sort.Strings(idxs)
			tags = append(tags, fmt.Sprintf(`indexed:"%s"`, strings.Join(idxs, ",")))
		}

		fields = append(fields, field{
			Name:    export(col),
			Type:    goType,
			TagText: "`" + strings.Join(tags, " ") + "`",
		})
	}

	// Build header summaries (pretty comments)
	header := buildHeaderSummary(dbName, schemaName, t)

	data := tmplData{
		Package:   "generated_models",
		Header:    header,
		Struct:    export(t.Name),
		Fields:    fields,
		Imports:   inferImports(fields),
		DbName:    dbName,
		Schema:    schemaName,
		TableName: t.Name,
	}

	var b strings.Builder
	if err := fileTmpl.Execute(&b, data); err != nil {
		return "", err
	}
	return b.String(), nil
}

// ---------- helpers ----------

type field struct {
	Name    string
	Type    string
	TagText string
}

type tmplData struct {
	Package   string
	Header    string
	Struct    string
	Fields    []field
	Imports   []string
	DbName    string
	Schema    string
	TableName string
}

func export(s string) string {
	// simple PascalCase
	s = strings.TrimSpace(s)
	parts := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == ' ' || r == '-' || r == '.'
	})
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + strings.ToLower(parts[i][1:])
	}
	return strings.Join(parts, "")
}

func sanitize(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(strings.ToLower(s), ".", "_"), "-", "_")
}

func mapPgTypeToGo(dt string) string {
	// best-effort mapping; feel free to extend
	switch strings.ToLower(dt) {
	case "uuid":
		return "string"
	case "text", "varchar", "character varying", "citext":
		return "string"
	case "bool", "boolean":
		return "bool"
	case "int2", "smallint":
		return "int16"
	case "int4", "integer":
		return "int"
	case "int8", "bigint":
		return "int64"
	case "numeric", "decimal":
		return "string" // avoid float precision; caller can parse to big.Rat/decimal
	case "float4", "real":
		return "float32"
	case "float8", "double precision":
		return "float64"
	case "date":
		return "time.Time"
	case "timestamp", "timestamp without time zone", "timestamp with time zone", "timestamptz":
		return "time.Time"
	case "json", "jsonb":
		return "[]byte"
	case "bytea":
		return "[]byte"
	default:
		// arrays or unrecognized types → string by default
		if strings.HasSuffix(dt, "[]") {
			return "[]string"
		}
		return "string"
	}
}

func inferImports(fields []field) []string {
	needTime := false
	for _, f := range fields {
		if f.Type == "time.Time" {
			needTime = true
			break
		}
	}
	var imps []string
	if needTime {
		imps = append(imps, "time")
	}
	return imps
}

func buildHeaderSummary(dbName, schema string, t *models.TableSnapshot) string {
	lines := []string{
		"Code generated by GoMetaSync. DO NOT EDIT.",
		fmt.Sprintf("Database: %s  Schema: %s  Table: %s", dbName, schema, t.Name),
	}

	// PK
	if len(t.PrimaryKey) > 0 {
		lines = append(lines, "PK: "+strings.Join(t.PrimaryKey, ", "))
	}

	// UNIQUE
	if len(t.UniqueConstraints) > 0 {
		keys := make([]string, 0, len(t.UniqueConstraints))
		for name, cols := range t.UniqueConstraints {
			keys = append(keys, fmt.Sprintf("%s(%s)", name, strings.Join(cols, ",")))
		}
		sort.Strings(keys)
		lines = append(lines, "UNIQUE: "+strings.Join(keys, " | "))
	}

	// CHECK (names only, to keep header compact)
	if len(t.CheckConstraints) > 0 {
		names := make([]string, 0, len(t.CheckConstraints))
		for name := range t.CheckConstraints {
			names = append(names, name)
		}
		sort.Strings(names)
		lines = append(lines, "CHECK: "+strings.Join(names, ", "))
	}

	// FKs
	if len(t.ForeignKeys) > 0 {
		keys := make([]string, 0, len(t.ForeignKeys))
		for name, fk := range t.ForeignKeys {
			keys = append(keys, fmt.Sprintf("%s(%s)->%s.%s(%s)",
				name, strings.Join(fk.Columns, ","),
				fk.RefSchema, fk.RefTable, strings.Join(fk.RefColumns, ",")))
		}
		sort.Strings(keys)
		lines = append(lines, "FK: "+strings.Join(keys, " | "))
	}

	// Indexes
	if len(t.Indexes) > 0 {
		keys := make([]string, 0, len(t.Indexes))
		for name, idx := range t.Indexes {
			prefix := "IDX"
			if idx.Unique {
				prefix = "UNIQ_IDX"
			}
			if len(idx.Columns) > 0 {
				keys = append(keys, fmt.Sprintf("%s %s(%s)", prefix, name, strings.Join(idx.Columns, ",")))
			} else {
				keys = append(keys, fmt.Sprintf("%s %s", prefix, name))
			}
		}
		sort.Strings(keys)
		lines = append(lines, "INDEX: "+strings.Join(keys, " | "))
	}

	return strings.Join(lines, "\n// ")
}

var fileTmpl = template.Must(template.New("file").Parse(`// {{.Header}}

package {{.Package}}

{{- if .Imports }}
import (
{{- range .Imports }}
	"{{.}}"
{{- end }}
)
{{- end }}

// {{.Struct}} maps to {{.DbName}}.{{.Schema}}.{{.TableName}}
type {{.Struct}} struct {
{{- range .Fields }}
	{{ .Name }} {{ .Type }} {{ .TagText }}
{{- end }}
}
`))
