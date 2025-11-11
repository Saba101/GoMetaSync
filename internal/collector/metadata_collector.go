package collector

import (
	"context"
	"time"

	"GoMetaSync.com/internal/models"
	"github.com/jackc/pgx/v5"
)

func CollectSnapshot(env string, dbs map[string]string) (*models.Snapshot, error) {
	snap := &models.Snapshot{
		Timestamp: time.Now(),
		Env:       env,
		Databases: map[string]models.DatabaseSnapshot{},
	}

	ctx := context.Background()

	for name, dsn := range dbs {
		conn, err := pgx.Connect(ctx, dsn)
		if err != nil {
			return nil, err
		}

		dbSnap := models.DatabaseSnapshot{
			DBName:  name,
			Schemas: map[string]models.SchemaSnapshot{},
		}

		// Fetch schemas
		rows, err := conn.Query(ctx, `
            SELECT schema_name
            FROM information_schema.schemata
            WHERE schema_name NOT IN ('pg_catalog','information_schema')
        `)
		if err != nil {
			return nil, err
		}

		for rows.Next() {
			var schema string
			_ = rows.Scan(&schema)
			dbSnap.Schemas[schema] = models.SchemaSnapshot{
				Name:   schema,
				Tables: map[string]models.TableSnapshot{},
			}
		}
		rows.Close()

		// Fetch tables + columns
		for schema := range dbSnap.Schemas {
			rows, err := conn.Query(ctx, `
                SELECT table_name, column_name, data_type
                FROM information_schema.columns
                WHERE table_schema = $1
                ORDER BY table_name, ordinal_position
            `, schema)
			if err != nil {
				return nil, err
			}

			for rows.Next() {
				var table, column, dtype string
				_ = rows.Scan(&table, &column, &dtype)

				tblSnap, exists := dbSnap.Schemas[schema].Tables[table]
				if !exists {
					tblSnap = models.TableSnapshot{
						Name:    table,
						Columns: map[string]string{},
					}
				}
				tblSnap.Columns[column] = dtype
				dbSnap.Schemas[schema].Tables[table] = tblSnap
			}
			rows.Close()
		}

		snap.Databases[name] = dbSnap
		conn.Close(ctx)
	}

	return snap, nil
}
