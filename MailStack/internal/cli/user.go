package cli

import (
	"fmt"

	"github.com/mailstack/mailstack/internal/config"
	"github.com/mailstack/mailstack/internal/database"
	"github.com/spf13/cobra"
)

func userCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Manage mail users",
		Long:  `Add, remove, list, and modify mail users.`,
	}

	cmd.AddCommand(userAddCmd())
	cmd.AddCommand(userDeleteCmd())
	cmd.AddCommand(userListCmd())
	cmd.AddCommand(userPasswordCmd())

	return cmd
}

func userAddCmd() *cobra.Command {
	var quota int64
	var password string

	cmd := &cobra.Command{
		Use:   "add <email>",
		Short: "Add a new mail user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			db, err := database.Connect(cfg.Database)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := db.AddUser(email, password, quota); err != nil {
				return fmt.Errorf("failed to add user: %w", err)
			}

			fmt.Printf("✅ User %s added successfully\n", email)
			return nil
		},
	}

	cmd.Flags().StringVarP(&password, "password", "p", "", "user password (will prompt if not provided)")
	cmd.Flags().Int64VarP(&quota, "quota", "q", 1000000000, "mailbox quota in bytes")

	return cmd
}

func userDeleteCmd() *cobra.Command {
	var removeMailbox bool

	cmd := &cobra.Command{
		Use:   "delete <email>",
		Short: "Delete a mail user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			db, err := database.Connect(cfg.Database)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := db.DeleteUser(email, removeMailbox); err != nil {
				return fmt.Errorf("failed to delete user: %w", err)
			}

			fmt.Printf("✅ User %s deleted successfully\n", email)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&removeMailbox, "remove-mailbox", "r", false, "also remove user's mailbox data")

	return cmd
}

func userListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all mail users",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			db, err := database.Connect(cfg.Database)
			if err != nil {
				return err
			}
			defer db.Close()

			users, err := db.ListUsers()
			if err != nil {
				return err
			}

			fmt.Println("📧 Mail Users:")
			for _, user := range users {
				fmt.Printf("  - %s (quota: %d MB)\n", user.Email, user.Quota/(1024*1024))
			}

			return nil
		},
	}
}

func userPasswordCmd() *cobra.Command {
	var password string

	cmd := &cobra.Command{
		Use:   "password <email>",
		Short: "Change user password",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			db, err := database.Connect(cfg.Database)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := db.ChangePassword(email, password); err != nil {
				return fmt.Errorf("failed to change password: %w", err)
			}

			fmt.Printf("✅ Password changed for %s\n", email)
			return nil
		},
	}

	cmd.Flags().StringVarP(&password, "password", "p", "", "new password (will prompt if not provided)")

	return cmd
}
