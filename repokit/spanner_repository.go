package repokit

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	"cloud.google.com/go/spanner"
	"google.golang.org/api/iterator"
)

// SpannerRepository provides a generic, type-safe repository implementation
// for Cloud Spanner. It supports CRUD, transactional operations, pagination,
// and key-returning inserts.
//
// T represents the domain entity mapped to a Spanner table.
type SpannerRepository[T any] struct {
	client          *spanner.Client
	tableName       string
	primaryKeys     []string
	rowMapper       func(*spanner.Row) (T, error)
	mutationBuilder func(entity T) *spanner.Mutation
}

func buildColumnList(columns []string) string {
	if len(columns) == 0 {
		return "*"
	}
	return strings.Join(columns, ", ")
}

func buildWhereClause(keys []string) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = fmt.Sprintf("%s = @%s", k, k)
	}
	return strings.Join(parts, " AND ")
}

func structToMap(key interface{}) (map[string]interface{}, error) {
	v := reflect.ValueOf(key)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil, fmt.Errorf("key precisa ser struct, recebido: %T", key)
	}

	t := v.Type()
	result := make(map[string]interface{}, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		field := t.Field(i)
		name := field.Tag.Get("spanner") // usamos tag `spanner:"col_name"`
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		result[name] = v.Field(i).Interface()
	}
	return result, nil
}

// FindByID fetches a single entity by its primary key.
func (r *SpannerRepository[T]) FindByID(ctx context.Context, key interface{}, columns []string) (T, bool, error) {
	var entity T

	params, err := structToMap(key)
	if err != nil {
		return entity, false, err
	}

	where := buildWhereClause(r.primaryKeys)
	stmt := spanner.Statement{
		SQL:    fmt.Sprintf("SELECT %s FROM %s WHERE %s", buildColumnList(columns), r.tableName, where),
		Params: params,
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return entity, false, err
	}

	entity, err = r.rowMapper(row)
	if err != nil {
		return entity, false, err
	}
	return entity, true, nil
}

// FindAll retrieves all rows from the table.
func (r *SpannerRepository[T]) FindAll(ctx context.Context, columns []string) ([]T, error) {
	stmt := spanner.Statement{
		SQL: fmt.Sprintf("SELECT %s FROM %s", buildColumnList(columns), r.tableName),
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []T
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		entity, err := r.rowMapper(row)
		if err != nil {
			return nil, err
		}
		results = append(results, entity)
	}
	return results, nil
}

// FindByIDs fetches multiple entities by their primary keys.
func (r *SpannerRepository[T]) FindByIDs(ctx context.Context, keys []interface{}, columns []string) ([]T, error) {
	var spannerKeys []spanner.Key
	for _, k := range keys {
		params, err := structToMap(k)
		if err != nil {
			return nil, err
		}
		values := make([]interface{}, len(r.primaryKeys))
		for i, col := range r.primaryKeys {
			values[i] = params[col]
		}
		spannerKeys = append(spannerKeys, spanner.Key(values))
	}

	keySet := spanner.KeySetFromKeys(spannerKeys...)

	iter := r.client.Single().Read(ctx, r.tableName, keySet, columns)
	defer iter.Stop()

	var results []T
	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}

		entity, err := r.rowMapper(row)
		if err != nil {
			return nil, err
		}
		results = append(results, entity)
	}
	return results, nil
}

// Save performs an upsert (insert or update) using a mutation.
func (r *SpannerRepository[T]) Save(ctx context.Context, entity T) error {
	m := r.mutationBuilder(entity)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// Update updates an entity in the table.
func (r *SpannerRepository[T]) Update(ctx context.Context, entity T) error {
	m := r.mutationBuilder(entity)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// Delete removes an entity from the table by primary key.
func (r *SpannerRepository[T]) Delete(ctx context.Context, key interface{}) error {
	params, err := structToMap(key)
	if err != nil {
		return err
	}

	values := make([]interface{}, len(r.primaryKeys))
	for i, k := range r.primaryKeys {
		values[i] = params[k]
	}

	m := spanner.Delete(r.tableName, spanner.Key(values))
	_, err = r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

// SaveReturningKey inserts a row using DML and returns the generated primary key.
func (r *SpannerRepository[T]) SaveReturningKey(
	ctx context.Context,
	insertSQL string,
	params map[string]interface{},
	dest interface{},
) error {
	_, err := r.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		stmt := spanner.Statement{SQL: insertSQL, Params: params}
		iter := txn.Query(ctx, stmt)
		defer iter.Stop()

		row, err := iter.Next()
		if err != nil {
			return err
		}

		if err := row.Column(0, dest); err != nil {
			return err
		}
		return nil
	})
	return err
}

// SaveReturningKeyTx is the transactional version of SaveReturningKey.
func (r *SpannerRepository[T]) SaveReturningKeyTx(
	ctx context.Context,
	txn *spanner.ReadWriteTransaction,
	insertSQL string,
	params map[string]interface{},
	dest interface{},
) error {
	stmt := spanner.Statement{SQL: insertSQL, Params: params}
	iter := txn.Query(ctx, stmt)
	defer iter.Stop()

	row, err := iter.Next()
	if err != nil {
		return err
	}

	if err := row.Column(0, dest); err != nil {
		return err
	}
	return nil
}

// Exists checks whether an entity exists by primary key.
func (r *SpannerRepository[T]) Exists(ctx context.Context, key interface{}) (bool, error) {
	_, found, err := r.FindByID(ctx, key, r.primaryKeys)
	return found, err
}

// SaveTx performs an upsert inside a transaction.
func (r *SpannerRepository[T]) SaveTx(tx Transaction, entity T) error {
	stx, ok := tx.(*SpannerTransaction)
	if !ok {
		return fmt.Errorf("invalid transaction type")
	}
	m := r.mutationBuilder(entity)
	return stx.ReadWriteTransaction().BufferWrite([]*spanner.Mutation{m})
}

// DeleteTx removes an entity inside a transaction.
func (r *SpannerRepository[T]) DeleteTx(tx Transaction, key interface{}) error {
	stx, ok := tx.(*SpannerTransaction)
	if !ok {
		return fmt.Errorf("invalid transaction type")
	}
	params, err := structToMap(key)
	if err != nil {
		return err
	}

	values := make([]interface{}, len(r.primaryKeys))
	for i, k := range r.primaryKeys {
		values[i] = params[k]
	}

	m := spanner.Delete(r.tableName, spanner.Key(values))
	return stx.ReadWriteTransaction().BufferWrite([]*spanner.Mutation{m})
}

// UpdateTx updates an entity inside a transaction.
func (r *SpannerRepository[T]) UpdateTx(tx Transaction, entity T) error {
	stx, ok := tx.(*SpannerTransaction)
	if !ok {
		return fmt.Errorf("invalid transaction type")
	}
	m := r.mutationBuilder(entity)
	return stx.ReadWriteTransaction().BufferWrite([]*spanner.Mutation{m})
}

// FindPage fetches entities with cursor-based pagination.
// pageToken should be the last seen primary key from a previous page.
func (r *SpannerRepository[T]) FindPage(
	ctx context.Context,
	pageSize int,
	pageToken interface{},
	columns []string,
) ([]T, interface{}, error) {
	var stmt spanner.Statement

	if pageToken == nil || pageToken == "" {
		stmt = spanner.Statement{
			SQL: fmt.Sprintf(`SELECT %s FROM %s ORDER BY %s LIMIT @limit`,
				buildColumnList(columns), r.tableName, strings.Join(r.primaryKeys, ", ")),
			Params: map[string]interface{}{
				"limit": pageSize,
			},
		}
	} else {
		stmt = spanner.Statement{
			SQL: fmt.Sprintf(`SELECT %s FROM %s WHERE %s > @pageToken ORDER BY %s LIMIT @limit`,
				buildColumnList(columns), r.tableName, r.primaryKeys[0],
				strings.Join(r.primaryKeys, ", ")),
			Params: map[string]interface{}{
				"pageToken": pageToken,
				"limit":     pageSize,
			},
		}
	}

	iter := r.client.Single().Query(ctx, stmt)
	defer iter.Stop()

	var results []T
	var lastKey interface{}

	for {
		row, err := iter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, nil, err
		}

		entity, err := r.rowMapper(row)
		if err != nil {
			return nil, nil, err
		}
		results = append(results, entity)

		if err := row.ColumnByName(r.primaryKeys[0], &lastKey); err != nil {
			return nil, nil, err
		}
	}

	return results, lastKey, nil
}

// NewBaseRepository creates a new generic SpannerRepository for a given entity type.
//
// Parameters:
//   - client: the Cloud Spanner client used to execute queries and mutations.
//   - tableName: the name of the Spanner table associated with the entity.
//   - primaryKeys: a slice of column names representing the primary key(s) of the table,
//     in the exact order defined in the schema.
//   - rowMapper: a function that maps a Spanner row (*spanner.Row) into an instance of T.
//     It is responsible for decoding database columns into the Go struct fields.
//   - mutationBuilder: a function that builds a Spanner mutation (*spanner.Mutation) from
//     an entity T. Typically, this defines how inserts/updates (UPSERT) are performed.
//
// Returns:
//   - *SpannerRepository[T]: a fully configured repository ready to perform CRUD,
//     transactional operations, and queries on the target table.
//
// Example:
//
//	repo := repokit.NewBaseRepository[User](
//	    client,
//	    "Users",
//	    []string{"user_id"},
//	    userRowMapper,
//	    userMutationBuilder,
//	)
//
//	user, found, err := repo.FindByID(ctx, UserKey{ID: "123"}, []string{"user_id", "email"})
func NewBaseRepository[T any](
	client *spanner.Client,
	tableName string,
	primaryKeys []string,
	rowMapper func(*spanner.Row) (T, error),
	mutationBuilder func(entity T) *spanner.Mutation,
) *SpannerRepository[T] {
	return &SpannerRepository[T]{
		client:          client,
		tableName:       tableName,
		primaryKeys:     primaryKeys,
		rowMapper:       rowMapper,
		mutationBuilder: mutationBuilder,
	}
}
