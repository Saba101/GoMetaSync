# Metadata Sync Prototype (Go)

This is a lightweight prototype to detect schema drift across multiple PostgreSQL databases in a single environment (DEV, QA, STAGE, PROD).

## ✅ What It Does
- Connects to multiple databases
- Reads all schemas, tables, and columns
- Saves a metadata snapshot as JSON
- Compares the new snapshot with an older one
- Shows differences:
  - ✅ new tables
  - ✅ removed tables
  - ✅ added or removed columns
  - ✅ data type changes

Example output:
✅ New table: dataflow_server.public.schedule
✅ New column: dataflow_server.public.job_logs.error_message (text)
⚠️ Type changed: datasource_server.public.users.age (integer → text)
❌ Column dropped: datasource_server.public.data_source.secret_key


---
## ✅ Project Structure

database-drift-syncronizer/

├── cmd/main.go # CLI entrypoint

├── internal/

│ ├── config/ # YAML config loader

│ ├── collector/ # Database metadata reader

│ ├── snapshot/ # Save/load JSON snapshots + diff

│ └── models/ # Snapshot structs

├── configs/ # Database connection configs

└── snapshots/ # Generated snapshot JSON files

---

## ✅ Configurations
`configs/dev.yml`

```yaml
env: LOCAL
databases:
  - name: dataflow_server
    # Option A — DSN:
    dsn: postgres://admin:<password>@localhost:<port>/<database_name>?sslmode=disable

    # Option B — explicit config:
    host: "localhost"
    port: <port>
    username: "admin"
    password: "<password>"
    database: "<database_name>"
    synchronize: "false"
    ssl: false
    rejectUnauthorized: false
    dbType: "postgres"
```

---
---
## Commands:
### Generate a Snapshot of Database:
```bash
    go run cmd/main.go \
    --mode snapshot \
    --config configs/dev.yml \
    --new snapshots/dev-1.json
```
### Detect Schema/Metadata Drift:
```bash
go run cmd/main.go \
  --mode diff \
  --old snapshots/dev-1.json \
  --new snapshots/dev-2.json
```


### Generate GO models from json file:
```bash
go run cmd/main.go \
  --mode generate \
  --new snapshots/dev-2.json \
  --out generated_models
  ```

  go run cmd/main.go --mode generate  --new snapshots/dataflow-local-1.json --out generated_models