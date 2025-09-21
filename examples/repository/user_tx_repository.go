package repository

import (
	"cloud.google.com/go/spanner"
	"context"
	"fmt"
	"github.com/Waelson/go-spanner-repo/examples/domain"
	"github.com/Waelson/go-spanner-repo/repokit"
)

type userTxRepository struct {
	base *repokit.SpannerRepository[domain.User]
}

type UserTxRepository interface {
	SaveTx(ctx context.Context, tx repokit.Transaction, user domain.User) (domain.User, error)
}

func (u *userTxRepository) SaveTx(ctx context.Context, tx repokit.Transaction, user domain.User) (domain.User, error) {
	stx, ok := tx.(*repokit.SpannerTransaction)
	if !ok {
		return user, fmt.Errorf("invalid transaction type")
	}
	err := u.base.SaveTx(stx.ReadWriteTransaction(), user)
	return user, err
}

// NewUserTxRepository creates a new NewUserTxRepository backed by Spanner.
func NewUserTxRepository(client *spanner.Client) UserTxRepository {
	base := repokit.NewBaseRepository[domain.User](
		client,
		userTable,  // Table name
		primaryKey, // Primary key columns
		userRowMapper,
		userMutationBuilder,
	)
	return &userTxRepository{base: base}
}
