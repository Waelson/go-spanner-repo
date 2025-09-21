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

func (r *SpannerRepository[T]) Save(ctx context.Context, entity T) error {
	m := r.mutationBuilder(entity)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

func (r *SpannerRepository[T]) Update(ctx context.Context, entity T) error {
	m := r.mutationBuilder(entity)
	_, err := r.client.Apply(ctx, []*spanner.Mutation{m})
	return err
}

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

func (r *SpannerRepository[T]) Exists(ctx context.Context, key interface{}) (bool, error) {
	_, found, err := r.FindByID(ctx, key, r.primaryKeys)
	return found, err
}

func (r *SpannerRepository[T]) SaveTx(txn *spanner.ReadWriteTransaction, entity T) error {
	m := r.mutationBuilder(entity)
	return txn.BufferWrite([]*spanner.Mutation{m})
}

func (r *SpannerRepository[T]) DeleteTx(txn *spanner.ReadWriteTransaction, key interface{}) error {
	params, err := structToMap(key)
	if err != nil {
		return err
	}

	values := make([]interface{}, len(r.primaryKeys))
	for i, k := range r.primaryKeys {
		values[i] = params[k]
	}

	m := spanner.Delete(r.tableName, spanner.Key(values))
	return txn.BufferWrite([]*spanner.Mutation{m})
}

// UpdateTx idem SaveTx
func (r *SpannerRepository[T]) UpdateTx(txn *spanner.ReadWriteTransaction, entity T) error {
	m := r.mutationBuilder(entity)
	return txn.BufferWrite([]*spanner.Mutation{m})
}

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
