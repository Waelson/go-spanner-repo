package repository

import (
	"cloud.google.com/go/spanner"
	"context"
	"github.com/Waelson/go-spanner-repo/examples/domain"
	"github.com/Waelson/go-spanner-repo/repokit"
)

// userTxRepository provides transactional operations for the User entity.
type userTxRepository struct {
	base *repokit.SpannerRepository[domain.User]
}

// UserTxRepository defines the transactional operations available for User.
// All methods must be executed inside a transaction managed by repokit.
type UserTxRepository interface {
	// SaveTx inserts or updates a User within a transaction.
	SaveTx(ctx context.Context, tx repokit.Transaction, user domain.User) (domain.User, error)

	// DeleteTx removes a User by ID within a transaction.
	DeleteTx(ctx context.Context, tx repokit.Transaction, userID string) error

	// UpdateTx updates a User within a transaction.
	UpdateTx(ctx context.Context, tx repokit.Transaction, user domain.User) error
}

func (u *userTxRepository) SaveTx(ctx context.Context, tx repokit.Transaction, user domain.User) (domain.User, error) {
	err := u.base.SaveTx(tx, user)
	return user, err
}

func (u *userTxRepository) DeleteTx(ctx context.Context, tx repokit.Transaction, userID string) error {
	return u.base.DeleteTx(tx, userID)
}

func (u *userTxRepository) UpdateTx(ctx context.Context, tx repokit.Transaction, user domain.User) error {
	return u.base.UpdateTx(tx, user)
}

// NewUserTxRepository creates a new transactional User repository backed by Spanner.
// It requires a spanner.Client and internally uses SpannerRepository for persistence.
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
