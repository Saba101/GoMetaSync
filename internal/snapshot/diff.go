package snapshot

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Saba101/GoMetaSync/internal/models"
)

func Diff(oldSnap, newSnap *models.Snapshot) {
	for db, newDB := range newSnap.Databases {
		oldDB := oldSnap.Databases[db]

		// Schemas
		for schema := range newDB.Schemas {
			if _, ok := oldDB.Schemas[schema]; !ok {
				fmt.Printf("✅ New schema: %s.%s\n", db, schema)
			}
		}
		for schema := range oldDB.Schemas {
			if _, ok := newDB.Schemas[schema]; !ok {
				fmt.Printf("❌ Schema dropped: %s.%s\n", db, schema)
			}
		}

		// Tables
		for schema, newSchema := range newDB.Schemas {
			oldSchema := oldDB.Schemas[schema]

			for tbl := range newSchema.Tables {
				if _, ok := oldSchema.Tables[tbl]; !ok {
					fmt.Printf("✅ New table: %s.%s.%s\n", db, schema, tbl)
				}
			}
			for tbl := range oldSchema.Tables {
				if _, ok := newSchema.Tables[tbl]; !ok {
					fmt.Printf("❌ Table dropped: %s.%s.%s\n", db, schema, tbl)
				}
			}

			// Per-table details
			for tbl, newTable := range newSchema.Tables {
				oldTable := oldSchema.Tables[tbl]

				// Columns
				for col, dt := range newTable.Columns {
					if _, ok := oldTable.Columns[col]; !ok {
						fmt.Printf("✅ New column: %s.%s.%s.%s (%s)\n", db, schema, tbl, col, dt)
					}
				}
				for col, oldT := range oldTable.Columns {
					nt, ok := newTable.Columns[col]
					if !ok {
						fmt.Printf("❌ Column dropped: %s.%s.%s.%s\n", db, schema, tbl, col)
					} else if nt != oldT {
						fmt.Printf("⚠️ Type changed: %s.%s.%s.%s (%s → %s)\n", db, schema, tbl, col, oldT, nt)
					}
				}

				// Primary key
				if !slices.Equal(oldTable.PrimaryKey, newTable.PrimaryKey) {
					if len(oldTable.PrimaryKey) == 0 && len(newTable.PrimaryKey) > 0 {
						fmt.Printf("✅ Primary key set: %s.%s.%s (%v)\n", db, schema, tbl, newTable.PrimaryKey)
					} else if len(newTable.PrimaryKey) == 0 && len(oldTable.PrimaryKey) > 0 {
						fmt.Printf("❌ Primary key dropped: %s.%s.%s (was %v)\n", db, schema, tbl, oldTable.PrimaryKey)
					} else {
						fmt.Printf("🔁 Primary key changed: %s.%s.%s (%v → %v)\n", db, schema, tbl, oldTable.PrimaryKey, newTable.PrimaryKey)
					}
				}

				// Unique constraints
				for name, cols := range newTable.UniqueConstraints {
					if _, ok := oldTable.UniqueConstraints[name]; !ok {
						fmt.Printf("✅ Unique constraint added: %s.%s.%s %s (%v)\n", db, schema, tbl, name, cols)
					}
				}
				for name, oldCols := range oldTable.UniqueConstraints {
					if newCols, ok := newTable.UniqueConstraints[name]; !ok {
						fmt.Printf("❌ Unique constraint dropped: %s.%s.%s %s\n", db, schema, tbl, name)
					} else if !slices.Equal(oldCols, newCols) {
						fmt.Printf("🔁 Unique constraint changed: %s.%s.%s %s (%v → %v)\n", db, schema, tbl, name, oldCols, newCols)
					}
				}

				// Check constraints
				for name, def := range newTable.CheckConstraints {
					if _, ok := oldTable.CheckConstraints[name]; !ok {
						fmt.Printf("✅ Check constraint added: %s.%s.%s %s\n", db, schema, tbl, name)
					} else if oldTable.CheckConstraints[name] != def {
						fmt.Printf("🔁 Check constraint changed: %s.%s.%s %s\n", db, schema, tbl, name)
					}
				}
				for name := range oldTable.CheckConstraints {
					if _, ok := newTable.CheckConstraints[name]; !ok {
						fmt.Printf("❌ Check constraint dropped: %s.%s.%s %s\n", db, schema, tbl, name)
					}
				}

				// Foreign keys
				for name, fk := range newTable.ForeignKeys {
					if _, ok := oldTable.ForeignKeys[name]; !ok {
						fmt.Printf("✅ Foreign key added: %s.%s.%s %s (%v → %s.%s %v)\n",
							db, schema, tbl, name, fk.Columns, fk.RefSchema, fk.RefTable, fk.RefColumns)
					}
				}
				for name, ofk := range oldTable.ForeignKeys {
					if nfk, ok := newTable.ForeignKeys[name]; !ok {
						fmt.Printf("❌ Foreign key dropped: %s.%s.%s %s\n", db, schema, tbl, name)
					} else {
						if !slices.Equal(ofk.Columns, nfk.Columns) ||
							ofk.RefSchema != nfk.RefSchema ||
							ofk.RefTable != nfk.RefTable ||
							!slices.Equal(ofk.RefColumns, nfk.RefColumns) ||
							ofk.UpdateRule != nfk.UpdateRule ||
							ofk.DeleteRule != nfk.DeleteRule {
							fmt.Printf("🔁 Foreign key changed: %s.%s.%s %s\n", db, schema, tbl, name)
						}
					}
				}

				// Indexes (by name)
				for name, idx := range newTable.Indexes {
					if _, ok := oldTable.Indexes[name]; !ok {
						fmt.Printf("✅ Index added: %s.%s.%s %s (unique=%v cols=%v)\n", db, schema, tbl, name, idx.Unique, idx.Columns)
					}
				}
				for name, oidx := range oldTable.Indexes {
					if nidx, ok := newTable.Indexes[name]; !ok {
						fmt.Printf("❌ Index dropped: %s.%s.%s %s\n", db, schema, tbl, name)
					} else if oidx.Unique != nidx.Unique || oidx.Definition != nidx.Definition || !slices.Equal(oidx.Columns, nidx.Columns) {
						fmt.Printf("🔁 Index changed: %s.%s.%s %s\n", db, schema, tbl, name)
					}
				}

				// sanity: any unknown keys?
				_ = maps.Clone(newTable.Columns) // keep compiler happy if using maps package
			}
		}
	}
}
