# Generic Repository for Spanner Database

A lightweight toolkit for building **repository layers** on top of **Google Cloud Spanner** using Go.  
It provides generic abstractions for CRUD, transactional operations, key-returning inserts, and pagination.

---

## ✨ Features

- Generic typed repository (`repokit`) for domain entities
- Flexible **row mapping** and **mutation building** functions
- Standard CRUD operations:
    - `FindByID`, `FindAll`, `FindByIDs`
    - `Save`, `Update`, `Delete`
- Key-returning inserts via DML (`SaveReturningKey`) — works with UUIDs or auto-incremented IDs
- Transaction support (`SaveTx`, `DeleteTx`, `UpdateTx`, `SaveReturningKeyTx`)
- Optional **cursor-based pagination** (no OFFSET required)

---

## 📦 Installation

```bash
go get github.com/Waelson/go-spanner-repo
```
---

## ⚠️ Notes
- Transactions: Use SaveTx, UpdateTx, DeleteTx inside a ReadWriteTransaction.
- Key returning inserts: Requires DML (INSERT ... THEN RETURN). Works with GENERATE_UUID() or sequence-backed INT64.
- Pagination: Implemented via cursor (pageToken = last seen PK). Avoids OFFSET for performance reasons.
- Composite PKs: Supported; ensure you pass the struct with all primary key fields.

---
## 🧪 Testing
```bash
go test ./repokit -v
```
---
## 🤝 Contributing
Contributions are welcome!
Please open an issue or submit a PR with clear description and tests.