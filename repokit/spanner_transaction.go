package repokit

import (
	"cloud.google.com/go/spanner"
	"context"
)

// SpannerTransaction is a wrapper around Cloud Spanner's
// *spanner.ReadWriteTransaction that also carries a context.
// It implements the Transaction interface so repositories can
// interact with transactions without depending directly on
// the Spanner client API.
type SpannerTransaction struct {
	ctx context.Context
	txn *spanner.ReadWriteTransaction
}

// Context returns the context associated with this transaction.
// It can be used to control deadlines or cancellation within
// the transaction scope.
func (t *SpannerTransaction) Context() context.Context {
	return t.ctx
}

// ReadWriteTransaction exposes the underlying Spanner transaction.
// This should typically only be used internally by repository
// implementations that need direct access to the Spanner API.
func (t *SpannerTransaction) ReadWriteTransaction() *spanner.ReadWriteTransaction {
	return t.txn
}

// SpannerTransactionManager manages execution of functions within
// a Cloud Spanner read-write transaction. It abstracts the Spanner
// client so that application code only deals with the generic
// Transaction interface.
type SpannerTransactionManager struct {
	client *spanner.Client
}

// NewSpannerTransactionManager creates a new transaction manager
// bound to the provided Spanner client. The manager is responsible
// for running functions inside read-write transactions.
func NewSpannerTransactionManager(client *spanner.Client) *SpannerTransactionManager {
	return &SpannerTransactionManager{client: client}
}

// RunInTransaction executes the given function inside a read-write
// transaction. If the function returns an error, the transaction is
// rolled back; otherwise, it is committed.
//
// Example:
//
//	txManager := repokit.NewSpannerTransactionManager(client)
//	err := txManager.RunInTransaction(ctx, func(tx repokit.Transaction) error {
//	    // Perform multiple repository operations atomically
//	    return nil
//	})
func (m *SpannerTransactionManager) RunInTransaction(
	ctx context.Context,
	fn func(transaction Transaction) error,
) error {
	_, err := m.client.ReadWriteTransaction(ctx, func(ctx context.Context, txn *spanner.ReadWriteTransaction) error {
		tx := &SpannerTransaction{ctx: ctx, txn: txn}
		return fn(tx)
	})
	return err
}
