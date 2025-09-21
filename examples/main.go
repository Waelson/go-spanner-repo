package main

import (
	"cloud.google.com/go/spanner"
	"context"
	"fmt"
	"github.com/Waelson/go-spanner-repo/examples/domain"
	"github.com/Waelson/go-spanner-repo/examples/repository"
	"github.com/Waelson/go-spanner-repo/repokit"
	"github.com/google/uuid"
	"log"
)

// main demonstrates how to use the Spanner repository abstraction
// for both non-transactional and transactional operations.
//
// It covers the full lifecycle of a User entity:
// - Insert
// - Find by ID
// - Update
// - Delete
//
// And also shows how to run multiple operations atomically within a transaction.
func main() {
	ctx := context.Background()

	// Create Spanner client
	spannerClient, err := createSpannerClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Initialize repository (non-transactional)
	userRepository := repository.NewUserNoTxRepository(spannerClient)

	// Create new user
	user := domain.User{
		UserID: uuid.New().String(), // Generate UUID for user ID
		Email:  "fake@email.com",
	}

	// ---- Non-transactional operations ---- //

	// Insert user
	user, err = userRepository.Save(ctx, user)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Inserted user: %+v\n", user)

	// Find by ID
	userDB, exists, err := userRepository.FindByID(ctx, user.UserID)
	if err != nil {
		log.Fatal(err)
	}
	if !exists {
		log.Println(fmt.Errorf("user not found"))
	}
	fmt.Printf("Found user: %+v\n", userDB)

	// Update user
	user.Email = "fake_tmp@email.com"
	err = userRepository.Update(ctx, user)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Updated user: %+v\n", user)

	// Delete user
	err = userRepository.Delete(ctx, user.UserID)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("User deleted")

	// ---- Transactional operations ---- //

	// Transaction manager is responsible for orchestrating transactions.
	txManager := repokit.NewSpannerTransactionManager(spannerClient)

	// Transactional repository version
	userTxRepository := repository.NewUserTxRepository(spannerClient)

	// Users to be inserted within the same transaction
	userTx1 := domain.User{
		UserID: uuid.New().String(),
		Email:  "usertx1@email.com",
	}
	userTx2 := domain.User{
		UserID: uuid.New().String(),
		Email:  "usertx2@email.com",
	}

	// Run multiple inserts atomically
	err = txManager.RunInTransaction(ctx, func(tx repokit.Transaction) error {
		userTx1, err = userTxRepository.SaveTx(ctx, tx, userTx1)
		if err != nil {
			return err
		}
		userTx2, err = userTxRepository.SaveTx(ctx, tx, userTx2)
		return err
	})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Transactional insert completed")
}

// createSpannerClient initializes a Cloud Spanner client given a project,
// instance, and database ID.
//
// Replace `your-project-id`, `your-instance-id`, and `your-database-id`
// with actual values before running.
func createSpannerClient(ctx context.Context) (*spanner.Client, error) {
	projectID := "your-project-id"
	instanceID := "your-instance-id"
	databaseID := "your-database-id"

	db := fmt.Sprintf("projects/%s/instances/%s/databases/%s", projectID, instanceID, databaseID)

	client, err := spanner.NewClient(ctx, db)
	if err != nil {
		return nil, err
	}
	return client, nil
}
