package cli

import (
	"fmt"

	"github.com/mailstack/mailstack/internal/config"
	"github.com/mailstack/mailstack/internal/database"
	"github.com/spf13/cobra"
)

func aliasCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alias",
		Short: "Manage email aliases",
		Long:  `Create, delete, and list email aliases and forwarding rules.`,
	}

	cmd.AddCommand(aliasAddCmd())
	cmd.AddCommand(aliasDeleteCmd())
	cmd.AddCommand(aliasListCmd())
	cmd.AddCommand(aliasShowCmd())

	return cmd
}

func aliasAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <alias-email> <destination>",
		Short: "Add a new email alias",
		Long: `Add a new email alias that forwards to one or more destinations.

Examples:
  mailstack alias add sales@example.com john@example.com
  mailstack alias add support@example.com john@example.com,jane@example.com
  mailstack alias add info@example.com external@gmail.com`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			email := args[0]
			destination := args[1]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			db, err := database.Connect(cfg.Database)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := db.AddAlias(email, destination); err != nil {
				return fmt.Errorf("failed to add alias: %w", err)
			}

			fmt.Printf("✅ Alias %s added successfully\n", email)
			fmt.Printf("   Forwards to: %s\n", destination)
			return nil
		},
	}
}

func aliasDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <alias-email>",
		Short: "Delete an email alias",
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

			if err := db.DeleteAlias(email); err != nil {
				return fmt.Errorf("failed to delete alias: %w", err)
			}

			fmt.Printf("✅ Alias %s deleted successfully\n", email)
			return nil
		},
	}
}

func aliasListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all email aliases",
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

			aliases, err := db.ListAliases()
			if err != nil {
				return err
			}

			if len(aliases) == 0 {
				fmt.Println("No aliases configured")
				return nil
			}

			fmt.Println("📧 Email Aliases:")
			for _, alias := range aliases {
				status := "✅"
				if !alias.Enabled {
					status = "❌"
				}
				fmt.Printf("  %s %s → %s\n", status, alias.Email, alias.Destination)
			}

			return nil
		},
	}
}

func aliasShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <alias-email>",
		Short: "Show details for a specific alias",
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

			alias, err := db.GetAlias(email)
			if err != nil {
				return err
			}

			fmt.Printf("📧 Alias: %s\n", alias.Email)
			fmt.Printf("   Destination: %s\n", alias.Destination)
			fmt.Printf("   Enabled: %v\n", alias.Enabled)

			return nil
		},
	}
}
