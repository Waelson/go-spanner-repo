package repokit

import (
	"cloud.google.com/go/spanner"
	"context"
)

type spannerTransaction struct {
	ctx context.Context
	txn *spanner.ReadWriteTransaction
}

func (t *spannerTransaction) Context() context.Context {
	return t.ctx
}

type SpannerTransactionManager struct {
	client *spanner.Client
}

func NewSpannerTransactionManager(client *spanner.Client) *SpannerTransactionManager {
	return &SpannerTransactionManager{client: client}
}

func (m *SpannerTransactionManager) RunInTransaction(ctx context.Context, fn func(transaction Transaction) error) error {
	_, err := m.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		tx := &spannerTransaction{ctx: ctx, txn: txn}
		return fn(tx)
	})
	return err
}
