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

## How to user
### Create an entity
```go
type User struct{
	UserID string
	Email string
}
```

### Implement specialized Repository
```go
type UserRepository struct {
    base *repokit.SpannerRepository[User]
}

func (u *UserRepository) Save(ctx context.Context, user User) (User, error) {
    err := u.base.Save(ctx, user)
    return user, err
}


func userRowMapper(row *spanner.Row) (User, error) {
    var u User
    err := row.Columns(&u.UserID, &u.Email)
    return u, err
}

func userMutationBuilder(u domain.User) *spanner.Mutation {
    return spanner.InsertOrUpdate(userTable, columns, []interface{}{u.UserID, u.Email})
}

func NewUserRepository(spannerClient *spanner.Client) UserRepository {
  base := repokit.NewBaseRepository[User](
    spannerClient, 
    "tb_users", 
    []string{"user_id"},
    userRowMapper,
    userMutationBuilder,
  )
  return &userTxRepository{base: base}	
}
```
### Consuming Repository
```go
func main(){
  ctx := context.Background()
  
  // Create Spanner client
  spannerClient, err := createSpannerClient(ctx)
  if err != nil {
    log.Fatal(err)
  }

  userRepository := repository.UserRepository(spannerClient)

  // Create new user
  user := domain.User{
    UserID: uuid.New().String(), // Generate UUID for user ID
    Email:  "fake@email.com",
  }
  
  
  // Insert user
  user, err = userRepository.Save(ctx, user)
  if err != nil {
    log.Fatal(err)
  }  
  
}
```

## 📖 API Overview

| Method                                     | Description                              |
| ------------------------------------------ | ---------------------------------------- |
| `FindByID(ctx, key, columns)`              | Fetch entity by primary key              |
| `FindAll(ctx, columns)`                    | Fetch all rows                           |
| `FindByIDs(ctx, keys, columns)`            | Lookup multiple entities by key          |
| `Save(ctx, entity)`                        | Insert or update (UPSERT)                |
| `Update(ctx, entity)`                      | Update entity                            |
| `Delete(ctx, key)`                         | Delete by primary key                    |
| `SaveReturningKey(ctx, sql, params, dest)` | Insert with DML and return generated key |
| `SaveTx`, `UpdateTx`, `DeleteTx`           | Transactional versions of mutations      |
| `SaveReturningKeyTx`                       | Transactional key-returning insert       |
| `Exists(ctx, key)`                         | Check if entity exists                   |

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