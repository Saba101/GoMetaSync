# 🚀 GoMetaSync

> **GoMetaSync** is an open-source tool written in Go that automatically detects database schema drift and keeps your Go structs in sync with live PostgreSQL databases.  
> It generates snapshots of your database metadata, detects differences between versions, and regenerates Go models dynamically — ensuring your application code always matches the actual database schema.

---

## Tech Stack
- Language: Go 1.23+
- Database: PostgreSQL
- Serialization: JSON, YAML
<!-- - Packages:
    pgx — PostgreSQL driver
yaml.v3 — YAML parsing -->

---

## 🧠 Overview

Database schemas evolve constantly — new columns, renamed tables, type changes, or even dropped fields.  
In fast-moving teams, these changes often **break existing application code** that relies on hardcoded models or outdated ORM mappings.

**GoMetaSync** solves this by introducing a **metadata synchronization layer**:

1. It periodically scans and snapshots your database schema.
2. It detects any drift between environments (e.g., DEV vs PROD).
3. It regenerates Go structs automatically from live metadata.

This makes GoMetaSync an excellent fit for:

- Teams managing **multiple environments** (DEV, QA, STAGE, PROD)
- Projects that rely on **code-generated models**
- Data engineering pipelines that need **metadata consistency**

---

## ⚙️ Core Features

✅ Detects **schema drift** across databases and environments  
✅ Generates **Go structs** directly from live database schemas  
✅ Produces **metadata snapshots** as versioned JSON files  
✅ Works with **PostgreSQL** out of the box  
✅ Extensible architecture — future support for MySQL, MSSQL, and SQLite  
✅ No ORM dependency — simple, portable, pure Go

---

## 🧩 Example Drift Output

✅ New table: dataflow_server.public.schedule

✅ New column: dataflow_server.public.job_logs.error_message (text)

⚠️ Type changed: datasource_server.public.users.age (integer → text)

❌ Column dropped: datasource_server.public.data_source.secret_key


---

## ✅ Project Structure

GoMetaSync/

├── cmd/main.go # CLI entrypoint

├── internal/

│ ├── config/ # YAML config loader

│ ├── collector/ # Database metadata reader

│ ├── snapshot/ # Save/load JSON snapshots + diff

│ ├── generated_models/ # Auto-generated Go structs

│ └── models/ # Snapshot structs

├── configs/ # Database connection configs

└── snapshots/ # Generated snapshot JSON files

---

## ⚙️ Configuration Example

`configs/dev.yml`

```yaml
env: LOCAL
databases:
  - name: dataflow_server
    # Option A — DSN:
    dsn: postgres://admin:<password>@localhost:<port>/<database>?sslmode=disable

    # Option B — explicit config:
    host: "localhost"
    port: <port>
    username: "admin"
    password: "<password>"
    database: "<database>"
    synchronize: false
    ssl: false
    rejectUnauthorized: false
```

---

## Installation

### 1️. Clone the repository
```bash
git clone https://github.com/Saba101/GoMetaSync.git
cd GoMetaSync
```

### 2️. Install dependencies
```
go mod tidy
```

### 3️. Build the CLI
```
go build -o metasynk cmd/main.go
./metasynk --help
```

---

## 🧭 Usage
### 1. Generate a Snapshot of Your Database
```
go run cmd/main.go \
  --mode snapshot \
  --config configs/dev.yml \
  --new snapshots/dev-1.json
```

### 2. Detect Schema Drift
```
go run cmd/main.go \
  --mode diff \
  --old snapshots/dev-1.json \
  --new snapshots/dev-2.json
```

### 3. Generate Go Structs
```
go run cmd/main.go \
  --mode generate \
  --new snapshots/dev-2.json \
  --out generated_models
```

---
## 🧪 Example Generated Struct
```go
// Code generated automatically. DO NOT EDIT.
package generated_models

import "time"

type Users struct {
    Id         string    `json:"id"`
    Name       string    `json:"name"`
    CreatedAt  time.Time `json:"created_at"`
}
```

---

## 🤝 Contributions
1. Pull requests, ideas, and feedback are welcome!
2. Fork the repository
3. Create a feature branch (feature/your-idea)
4. Commit your changes
5. Submit a pull request 🚀

---

## 🌟 Acknowledgements
- Inspired by schema management techniques in Superset, Metabase, and dbt
- Built for Go open-source community

----

## 📫 Connect
If you find this useful, please ⭐ the repo and share feedback!

For collaboration or feature ideas, reach out on LinkedIn: Saba Amin

---

## 🔖 Version
GoMetaSync **v1.0.0** — Initial public release (Schema Drift Detection & Struct Generation)