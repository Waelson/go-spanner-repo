package repository

import (
	"cloud.google.com/go/spanner"
	"context"
	"github.com/Waelson/go-spanner-repo/examples/domain"
	"github.com/Waelson/go-spanner-repo/repokit"
)

var (
	userTable  = "tb_users"
	primaryKey = []string{"user_id"}
	columns    = []string{"user_id", "email"}
)

// UserKey represents the primary key structure for the Users table.
type UserKey struct {
	ID string `spanner:"user_id"` //Primary key column name
}

// userNoTxRepository is a Spanner-backed implementation of UserNoTxRepository.
type userNoTxRepository struct {
	base *repokit.SpannerRepository[domain.User]
}

// UserNoTxRepository defines operations for managing User entities in Spanner.
type UserNoTxRepository interface {

	// FindByID retrieves a User by its primary key.
	FindByID(ctx context.Context, userID string) (domain.User, bool, error)

	// Save inserts or updates a User entity.
	Save(ctx context.Context, user domain.User) (domain.User, error)

	// Delete removes a User by its primary key.
	Delete(ctx context.Context, userID string) error

	// Update modifies an existing User entity.
	Update(ctx context.Context, user domain.User) error
}

// FindByID retrieves a user by ID from Spanner.
func (u *userNoTxRepository) FindByID(ctx context.Context, userID string) (domain.User, bool, error) {
	key := UserKey{ID: userID}
	return u.base.FindByID(ctx, key, columns)
}

// Save inserts or updates a user in Spanner.
func (u *userNoTxRepository) Save(ctx context.Context, user domain.User) (domain.User, error) {
	err := u.base.Save(ctx, user)
	return user, err
}

// Delete removes a user by ID from Spanner.
func (u *userNoTxRepository) Delete(ctx context.Context, userID string) error {
	key := UserKey{ID: userID}
	return u.base.Delete(ctx, key)
}

// Update modifies an existing user in Spanner.
func (u *userNoTxRepository) Update(ctx context.Context, user domain.User) error {
	return u.base.Update(ctx, user)
}

// userRowMapper converts a Spanner row into a User entity.
func userRowMapper(row *spanner.Row) (domain.User, error) {
	var u domain.User
	err := row.Columns(&u.UserID, &u.Email)
	return u, err
}

// userMutationBuilder builds a mutation for inserting or updating a User.
func userMutationBuilder(u domain.User) *spanner.Mutation {
	return spanner.InsertOrUpdate(userTable, columns, []interface{}{u.UserID, u.Email})
}

// NewUserNoTxRepository creates a new UserNoTxRepository backed by Spanner.
func NewUserNoTxRepository(client *spanner.Client) UserNoTxRepository {
	base := repokit.NewBaseRepository[domain.User](
		client,
		userTable,  // Table name
		primaryKey, // Primary key columns
		userRowMapper,
		userMutationBuilder,
	)
	return &userNoTxRepository{base: base}
}
