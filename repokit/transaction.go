package repokit

import "context"

type Transaction interface {
	Context() context.Context
}

type TransactionManager interface {
	RunInTransaction(ctx context.Context, fn func(tx Transaction) error) error
}
