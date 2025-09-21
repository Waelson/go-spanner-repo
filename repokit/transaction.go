package repokit

import "context"

// Transaction defines the minimal interface for an active database transaction.
// It provides access to the context bound to the transaction, which can be used
// to manage deadlines, cancellation, or propagate metadata.
type Transaction interface {
	// Context returns the context associated with this transaction.
	Context() context.Context
}

// TransactionManager abstracts the execution of operations inside a transaction.
// Implementations are responsible for starting, committing, and rolling back
// transactions depending on the function's outcome.
type TransactionManager interface {
	// RunInTransaction executes the given function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	// Otherwise, it is committed.
	//
	// Example:
	//
	//   err := txManager.RunInTransaction(ctx, func(tx repokit.Transaction) error {
	//       // perform repository operations atomically
	//       return nil
	//   })
	RunInTransaction(ctx context.Context, fn func(tx Transaction) error) error
}
