package snapshot

import (
	"fmt"

	"github.com/Saba101/GoMetaSync/internal/models"
)

func Diff(oldSnap, newSnap *models.Snapshot) {
	for db, newDB := range newSnap.Databases {
		oldDB := oldSnap.Databases[db]

		// NEW SCHEMA
		for schema := range newDB.Schemas {
			if _, exists := oldDB.Schemas[schema]; !exists {
				fmt.Printf("✅ New schema created: %s.%s\n", db, schema)
			}
		}

		// NEW TABLE / DROPPED TABLE
		for schema, newSchema := range newDB.Schemas {
			oldSchema := oldDB.Schemas[schema]

			for tbl := range newSchema.Tables {
				if _, exists := oldSchema.Tables[tbl]; !exists {
					fmt.Printf("✅ New table: %s.%s.%s\n", db, schema, tbl)
				}
			}

			for tbl := range oldSchema.Tables {
				if _, exists := newSchema.Tables[tbl]; !exists {
					fmt.Printf("❌ Table dropped: %s.%s.%s\n", db, schema, tbl)
				}
			}

			// COLUMN CHANGES
			for tbl, newTable := range newSchema.Tables {
				oldTable := oldSchema.Tables[tbl]

				// new columns
				for col, dtype := range newTable.Columns {
					if _, exists := oldTable.Columns[col]; !exists {
						fmt.Printf("✅ New column: %s.%s.%s.%s (%s)\n",
							db, schema, tbl, col, dtype)
					}
				}

				// dropped columns or type changes
				for col, oldType := range oldTable.Columns {
					newType, exists := newTable.Columns[col]
					if !exists {
						fmt.Printf("❌ Column dropped: %s.%s.%s.%s\n",
							db, schema, tbl, col)
						continue
					}

					if oldType != newType {
						fmt.Printf("⚠️ Type changed: %s.%s.%s.%s (%s → %s)\n",
							db, schema, tbl, col, oldType, newType)
					}
				}
			}
		}
	}
}
