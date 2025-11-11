package collector

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Saba101/GoMetaSync/internal/models"
	"github.com/jackc/pgx/v5"
)

func CollectSnapshot(env string, dbs map[string]string) (*models.Snapshot, error) {
	snap := &models.Snapshot{
		Timestamp: time.Now(),
		Env:       env,
		Databases: map[string]models.DatabaseSnapshot{},
	}
	ctx := context.Background()

	for dbName, dsn := range dbs {
		conn, err := pgx.Connect(ctx, dsn)
		if err != nil {
			return nil, err
		}

		dbSnap := models.DatabaseSnapshot{
			DBName:  dbName,
			Schemas: map[string]models.SchemaSnapshot{},
		}

		// ---- Schemas
		rows, err := conn.Query(ctx, `
			SELECT schema_name
			FROM information_schema.schemata
			WHERE schema_name NOT IN ('pg_catalog','information_schema')
			ORDER BY schema_name`)
		if err != nil {
			conn.Close(ctx)
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

		// ---- Tables & Columns
		for schema := range dbSnap.Schemas {
			// columns
			colRows, err := conn.Query(ctx, `
				SELECT table_name, column_name, data_type
				FROM information_schema.columns
				WHERE table_schema = $1
				ORDER BY table_name, ordinal_position`, schema)
			if err != nil {
				conn.Close(ctx)
				return nil, err
			}
			for colRows.Next() {
				var table, col, dtype string
				_ = colRows.Scan(&table, &col, &dtype)

				t, ok := dbSnap.Schemas[schema].Tables[table]
				if !ok {
					t = models.TableSnapshot{
						Name:              table,
						Columns:           map[string]string{},
						UniqueConstraints: map[string][]string{},
						CheckConstraints:  map[string]string{},
						ForeignKeys:       map[string]models.ForeignKey{},
						Indexes:           map[string]models.Index{},
					}
				}
				t.Columns[col] = dtype
				dbSnap.Schemas[schema].Tables[table] = t
			}
			colRows.Close()

			// primary keys & unique constraints (information_schema)
			if err := loadPKs(ctx, conn, schema, &dbSnap); err != nil {
				conn.Close(ctx)
				return nil, err
			}
			if err := loadUniqueConstraints(ctx, conn, schema, &dbSnap); err != nil {
				conn.Close(ctx)
				return nil, err
			}
			if err := loadCheckConstraints(ctx, conn, schema, &dbSnap); err != nil {
				conn.Close(ctx)
				return nil, err
			}
			// foreign keys
			if err := loadFKs(ctx, conn, schema, &dbSnap); err != nil {
				conn.Close(ctx)
				return nil, err
			}
			// indexes (via pg_indexes view)
			if err := loadIndexes(ctx, conn, schema, &dbSnap); err != nil {
				conn.Close(ctx)
				return nil, err
			}
		}

		snap.Databases[dbName] = dbSnap
		conn.Close(ctx)
	}

	return snap, nil
}

// ---------- helpers ----------

func loadPKs(ctx context.Context, conn *pgx.Conn, schema string, dbSnap *models.DatabaseSnapshot) error {
	rows, err := conn.Query(ctx, `
		SELECT tc.table_name, kcu.column_name, kcu.ordinal_position
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema    = kcu.table_schema
		 AND tc.table_name      = kcu.table_name
		WHERE tc.table_schema = $1 AND tc.constraint_type = 'PRIMARY KEY'
		ORDER BY tc.table_name, kcu.ordinal_position`, schema)
	if err != nil { return err }

	type entry struct{ table, col string; pos int32 }
	var es []entry
	for rows.Next() {
		var e entry
		_ = rows.Scan(&e.table, &e.col, &e.pos)
		es = append(es, e)
	}
	rows.Close()

	// assemble ordered list
	m := map[string][]struct{ pos int32; col string }{}
	for _, e := range es {
		m[e.table] = append(m[e.table], struct{ pos int32; col string }{e.pos, e.col})
	}
	for tbl, list := range m {
		sort.Slice(list, func(i, j int) bool { return list[i].pos < list[j].pos })
		cols := make([]string, len(list))
		for i, v := range list { cols[i] = v.col }
		t := dbSnap.Schemas[schema].Tables[tbl]
		t.PrimaryKey = cols
		dbSnap.Schemas[schema].Tables[tbl] = t
	}
	return nil
}

func loadUniqueConstraints(ctx context.Context, conn *pgx.Conn, schema string, dbSnap *models.DatabaseSnapshot) error {
	rows, err := conn.Query(ctx, `
		SELECT tc.table_name, tc.constraint_name, kcu.column_name, kcu.ordinal_position
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema    = kcu.table_schema
		 AND tc.table_name      = kcu.table_name
		WHERE tc.table_schema = $1 AND tc.constraint_type = 'UNIQUE'
		ORDER BY tc.table_name, tc.constraint_name, kcu.ordinal_position`, schema)
	if err != nil { return err }

	type entry struct{ table, cname, col string; pos int32 }
	var es []entry
	for rows.Next() {
		var e entry
		_ = rows.Scan(&e.table, &e.cname, &e.col, &e.pos)
		es = append(es, e)
	}
	rows.Close()

	group := map[string]map[string][]struct{ pos int32; col string }{} // table -> cname -> list
	for _, e := range es {
		if _, ok := group[e.table]; !ok { group[e.table] = map[string][]struct{ pos int32; col string }{} }
		group[e.table][e.cname] = append(group[e.table][e.cname], struct{ pos int32; col string }{e.pos, e.col})
	}
	for tbl, byC := range group {
		t := dbSnap.Schemas[schema].Tables[tbl]
		if t.UniqueConstraints == nil { t.UniqueConstraints = map[string][]string{} }
		for cname, list := range byC {
			sort.Slice(list, func(i, j int) bool { return list[i].pos < list[j].pos })
			out := make([]string, len(list))
			for i, v := range list { out[i] = v.col }
			t.UniqueConstraints[cname] = out
		}
		dbSnap.Schemas[schema].Tables[tbl] = t
	}
	return nil
}

func loadCheckConstraints(ctx context.Context, conn *pgx.Conn, schema string, dbSnap *models.DatabaseSnapshot) error {
	rows, err := conn.Query(ctx, `
		SELECT tc.table_name, tc.constraint_name, cc.check_clause
		FROM information_schema.table_constraints tc
		JOIN information_schema.check_constraints cc
		  ON tc.constraint_name = cc.constraint_name
		 AND tc.constraint_schema = cc.constraint_schema
		WHERE tc.table_schema = $1 AND tc.constraint_type = 'CHECK'
		ORDER BY tc.table_name, tc.constraint_name`, schema)
	if err != nil { return err }

	for rows.Next() {
		var tbl, cname, clause string
		_ = rows.Scan(&tbl, &cname, &clause)
		t := dbSnap.Schemas[schema].Tables[tbl]
		if t.CheckConstraints == nil { t.CheckConstraints = map[string]string{} }
		t.CheckConstraints[cname] = clause
		dbSnap.Schemas[schema].Tables[tbl] = t
	}
	rows.Close()
	return nil
}

func loadFKs(ctx context.Context, conn *pgx.Conn, schema string, dbSnap *models.DatabaseSnapshot) error {
	rows, err := conn.Query(ctx, `
		SELECT
		  tc.table_name,
		  tc.constraint_name,
		  kcu.column_name,
		  ccu.table_schema  AS ref_schema,
		  ccu.table_name    AS ref_table,
		  ccu.column_name   AS ref_column,
		  kcu.ordinal_position,
		  rc.update_rule,
		  rc.delete_rule
		FROM information_schema.table_constraints tc
		JOIN information_schema.key_column_usage kcu
		  ON tc.constraint_name = kcu.constraint_name
		 AND tc.table_schema    = kcu.table_schema
		JOIN information_schema.referential_constraints rc
		  ON tc.constraint_name = rc.constraint_name
		 AND tc.constraint_schema = rc.constraint_schema
		JOIN information_schema.constraint_column_usage ccu
		  ON rc.unique_constraint_name = ccu.constraint_name
		 AND rc.unique_constraint_schema = ccu.constraint_schema
		WHERE tc.table_schema = $1 AND tc.constraint_type = 'FOREIGN KEY'
		ORDER BY tc.table_name, tc.constraint_name, kcu.ordinal_position`, schema)
	if err != nil { return err }

	type row struct {
		table, cname, col, refSchema, refTable, refCol, updateRule, deleteRule string
		pos int32
	}
	var rs []row
	for rows.Next() {
		var r row
		_ = rows.Scan(&r.table, &r.cname, &r.col, &r.refSchema, &r.refTable, &r.refCol, &r.pos, &r.updateRule, &r.deleteRule)
		rs = append(rs, r)
	}
	rows.Close()

	type pair struct{ pos int32; col string }
	type refpair struct{ pos int32; col string }

	byTable := map[string]map[string]struct {
		cols []pair
		refc []refpair
		refS string
		refT string
		upd  string
		del  string
	}{}

	for _, r := range rs {
		if _, ok := byTable[r.table]; !ok {
			byTable[r.table] = map[string]struct {
				cols []pair; refc []refpair; refS, refT, upd, del string
			}{}
		}
		entry := byTable[r.table][r.cname]
		entry.cols = append(entry.cols, pair{r.pos, r.col})
		entry.refc = append(entry.refc, refpair{r.pos, r.refCol})
		entry.refS = r.refSchema
		entry.refT = r.refTable
		entry.upd = r.updateRule
		entry.del = r.deleteRule
		byTable[r.table][r.cname] = entry
	}

	for tbl, byC := range byTable {
		t := dbSnap.Schemas[schema].Tables[tbl]
		if t.ForeignKeys == nil { t.ForeignKeys = map[string]models.ForeignKey{} }
		for cname, v := range byC {
			sort.Slice(v.cols, func(i, j int) bool { return v.cols[i].pos < v.cols[j].pos })
			sort.Slice(v.refc, func(i, j int) bool { return v.refc[i].pos < v.refc[j].pos })
			lcols := make([]string, len(v.cols))
			rcols := make([]string, len(v.refc))
			for i := range v.cols { lcols[i] = v.cols[i].col }
			for i := range v.refc { rcols[i] = v.refc[i].col }
			t.ForeignKeys[cname] = models.ForeignKey{
				Name:       cname,
				Columns:    lcols,
				RefSchema:  v.refS,
				RefTable:   v.refT,
				RefColumns: rcols,
				UpdateRule: v.upd,
				DeleteRule: v.del,
			}
		}
		dbSnap.Schemas[schema].Tables[tbl] = t
	}
	return nil
}

func loadIndexes(ctx context.Context, conn *pgx.Conn, schema string, dbSnap *models.DatabaseSnapshot) error {
	rows, err := conn.Query(ctx, `
		SELECT tablename, indexname, indexdef
		FROM pg_catalog.pg_indexes
		WHERE schemaname = $1
		ORDER BY tablename, indexname`, schema)
	if err != nil { return err }

	for rows.Next() {
		var tbl, name, def string
		_ = rows.Scan(&tbl, &name, &def)
		t := dbSnap.Schemas[schema].Tables[tbl]
		if t.Indexes == nil { t.Indexes = map[string]models.Index{} }
		idx := models.Index{
			Name:       name,
			Unique:     strings.Contains(strings.ToUpper(def), "UNIQUE INDEX"),
			Definition: def,
			Columns:    parseIndexColumns(def),
		}
		t.Indexes[name] = idx
		dbSnap.Schemas[schema].Tables[tbl] = t
	}
	rows.Close()
	return nil
}

// crude but reliable parser for column list in pg_indexes.indexdef
// examples:
// CREATE UNIQUE INDEX idx ON public.users USING btree (email)
// CREATE INDEX idx2 ON public.users USING gin (tags) WHERE (deleted_at IS NULL)
var idxColsRe = regexp.MustCompile(`\(([^)]+)\)`)

func parseIndexColumns(def string) []string {
	m := idxColsRe.FindStringSubmatch(def)
	if len(m) < 2 { return nil }
	inside := m[1]
	// split by commas not considering expressions heavily; best-effort
	parts := strings.Split(inside, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		// drop sort order and NULLS options
		p = strings.Split(p, " ")[0]
		p = strings.Trim(p, `"`)
		if p != "" { out = append(out, p) }
	}
	return out
}
