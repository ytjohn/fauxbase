package cli

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ytjohn/fauxbase/pkg/appdb"
	"github.com/ytjohn/fauxbase/pkg/auth"
	_ "modernc.org/sqlite" // SQLite3 driver
)

const (
	sqliteDriver = "sqlite"
)

var (
	userEmail    string
	userPassword string
	userRole     string
)

func init() {
	rootCmd.AddCommand(userCmd)
	userCmd.AddCommand(createUserCmd)
	userCmd.AddCommand(changePasswordCmd)
	userCmd.AddCommand(deleteUserCmd)
	userCmd.AddCommand(listUsersCmd)

	// Create user flags
	createUserCmd.Flags().StringVarP(&userEmail, "email", "e", "", "User email")
	createUserCmd.Flags().StringVarP(&userPassword, "password", "p", "", "User password")
	createUserCmd.Flags().StringVarP(&userRole, "role", "r", "user", "User role (admin or user)")
	if err := createUserCmd.MarkFlagRequired("email"); err != nil {
		panic(fmt.Sprintf("failed to mark email flag as required: %v", err))
	}
	if err := createUserCmd.MarkFlagRequired("password"); err != nil {
		panic(fmt.Sprintf("failed to mark password flag as required: %v", err))
	}

	// Change password flags
	changePasswordCmd.Flags().StringVarP(&userEmail, "email", "e", "", "User email")
	changePasswordCmd.Flags().StringVarP(&userPassword, "password", "p", "", "New password")
	if err := changePasswordCmd.MarkFlagRequired("email"); err != nil {
		panic(fmt.Sprintf("failed to mark email flag as required: %v", err))
	}
	if err := changePasswordCmd.MarkFlagRequired("password"); err != nil {
		panic(fmt.Sprintf("failed to mark password flag as required: %v", err))
	}

	// Delete user flags
	deleteUserCmd.Flags().StringVarP(&userEmail, "email", "e", "", "User email")
	if err := deleteUserCmd.MarkFlagRequired("email"); err != nil {
		panic(fmt.Sprintf("failed to mark email flag as required: %v", err))
	}
}

var userCmd = &cobra.Command{
	Use:   "user",
	Short: "User management commands",
}

var createUserCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new user",
	Run: func(cmd *cobra.Command, args []string) {
		Log.Debug("opening database", "path", GlobalConfig.Database.Path)
		db, err := sql.Open(sqliteDriver, GlobalConfig.Database.Path)
		if err != nil {
			Log.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		queries := appdb.New(db)

		Log.Debug("hashing password for new user", "email", userEmail)
		hashedPassword, err := auth.HashPassword(userPassword, nil)
		if err != nil {
			Log.Error("failed to hash password", "error", err)
			os.Exit(1)
		}

		Log.Debug("creating user", "email", userEmail, "role", userRole)
		user, err := queries.CreateUser(cmd.Context(), appdb.CreateUserParams{
			Username: userEmail,
			Email:    userEmail,
			Password: hashedPassword,
			Role:     userRole,
		})
		if err != nil {
			Log.Error("failed to create user", "error", err)
			os.Exit(1)
		}

		Log.Info("user created successfully", "email", user.Email, "role", user.Role)
		fmt.Printf("Created user %s with role %s\n", user.Email, user.Role)
	},
}

var changePasswordCmd = &cobra.Command{
	Use:   "passwd",
	Short: "Change user password",
	Run: func(cmd *cobra.Command, args []string) {
		Log.Debug("opening database", "path", GlobalConfig.Database.Path)
		db, err := sql.Open(sqliteDriver, GlobalConfig.Database.Path)
		if err != nil {
			Log.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		queries := appdb.New(db)

		Log.Debug("hashing new password", "email", userEmail)
		hashedPassword, err := auth.HashPassword(userPassword, nil)
		if err != nil {
			Log.Error("failed to hash password", "error", err)
			os.Exit(1)
		}

		Log.Debug("updating password", "email", userEmail)
		err = queries.UpdateUserPassword(cmd.Context(), appdb.UpdateUserPasswordParams{
			Password: hashedPassword,
			Email:    userEmail,
		})
		if err != nil {
			Log.Error("failed to update password", "error", err)
			os.Exit(1)
		}

		Log.Info("password updated successfully", "email", userEmail)
		fmt.Printf("Updated password for user %s\n", userEmail)
	},
}

var deleteUserCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a user",
	Run: func(cmd *cobra.Command, args []string) {
		Log.Debug("opening database", "path", GlobalConfig.Database.Path)
		db, err := sql.Open(sqliteDriver, GlobalConfig.Database.Path)
		if err != nil {
			Log.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		queries := appdb.New(db)

		Log.Debug("deleting user", "email", userEmail)
		err = queries.DeleteUser(cmd.Context(), userEmail)
		if err != nil {
			Log.Error("failed to delete user", "error", err)
			os.Exit(1)
		}

		Log.Info("user deleted successfully", "email", userEmail)
		fmt.Printf("Deleted user %s\n", userEmail)
	},
}

var listUsersCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users",
	Run: func(cmd *cobra.Command, args []string) {
		Log.Debug("opening database", "path", GlobalConfig.Database.Path)
		db, err := sql.Open(sqliteDriver, GlobalConfig.Database.Path)
		if err != nil {
			Log.Error("failed to open database", "error", err)
			os.Exit(1)
		}
		defer db.Close()

		queries := appdb.New(db)

		Log.Debug("listing users")
		users, err := queries.ListUsers(cmd.Context())
		if err != nil {
			Log.Error("failed to list users", "error", err)
			os.Exit(1)
		}

		fmt.Printf("%-30s %-20s %-15s\n", "EMAIL", "ROLE", "LAST LOGIN")
		fmt.Println(strings.Repeat("-", 65))
		for _, user := range users {
			lastLogin := "Never"
			if user.Lastlogin.Valid {
				lastLogin = user.Lastlogin.Time.Format("2006-01-02 15:04:05")
			}
			fmt.Printf("%-30s %-20s %-15s\n", user.Email, user.Role, lastLogin)
		}
		Log.Debug("listed users", "count", len(users))
	},
}
