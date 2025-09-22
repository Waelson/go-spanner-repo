package repokit

import (
	"cloud.google.com/go/spanner"
)

// SpannerRepositoryBuilder provides a builder for constructing SpannerRepository instances.
type SpannerRepositoryBuilder[T any] struct {
	client      *spanner.Client
	tableName   string
	primaryKeys []string
	rowMapper   func(*spanner.Row) (T, error)
	mutation    func(entity T) *spanner.Mutation
}

// NewSpannerRepositoryBuilder initializes a new builder for SpannerRepository.
func NewSpannerRepositoryBuilder[T any]() *SpannerRepositoryBuilder[T] {
	return &SpannerRepositoryBuilder[T]{}
}

// WithClient sets the Cloud Spanner client.
func (b *SpannerRepositoryBuilder[T]) WithClient(client *spanner.Client) *SpannerRepositoryBuilder[T] {
	b.client = client
	return b
}

// WithTableName sets the table name.
func (b *SpannerRepositoryBuilder[T]) WithTableName(table string) *SpannerRepositoryBuilder[T] {
	b.tableName = table
	return b
}

// WithPrimaryKeys sets the primary key columns (in schema order).
func (b *SpannerRepositoryBuilder[T]) WithPrimaryKeys(keys []string) *SpannerRepositoryBuilder[T] {
	b.primaryKeys = keys
	return b
}

// WithRowMapper sets the row-to-entity mapper function.
func (b *SpannerRepositoryBuilder[T]) WithRowMapper(mapper func(*spanner.Row) (T, error)) *SpannerRepositoryBuilder[T] {
	b.rowMapper = mapper
	return b
}

// WithMutation sets the entity-to-mutation builder function.
func (b *SpannerRepositoryBuilder[T]) WithMutation(builder func(entity T) *spanner.Mutation) *SpannerRepositoryBuilder[T] {
	b.mutation = builder
	return b
}

// Build creates the SpannerRepository with the provided configuration.
func (b *SpannerRepositoryBuilder[T]) Build() *SpannerRepository[T] {
	return &SpannerRepository[T]{
		client:      b.client,
		tableName:   b.tableName,
		primaryKeys: b.primaryKeys,
		rowMapper:   b.rowMapper,
		mutation:    b.mutation,
	}
}
