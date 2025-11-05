package cli

import (
	"fmt"

	"github.com/mailstack/mailstack/internal/config"
	"github.com/mailstack/mailstack/internal/database"
	"github.com/spf13/cobra"
)

func domainCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domain",
		Short: "Manage mail domains",
		Long:  `Add, remove, and list mail domains.`,
	}

	cmd.AddCommand(domainAddCmd())
	cmd.AddCommand(domainDeleteCmd())
	cmd.AddCommand(domainListCmd())

	return cmd
}

func domainAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <domain>",
		Short: "Add a new mail domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			db, err := database.Connect(cfg.Database)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := db.AddDomain(domain); err != nil {
				return fmt.Errorf("failed to add domain: %w", err)
			}

			fmt.Printf("✅ Domain %s added successfully\n", domain)
			fmt.Println("\n📝 Don't forget to:")
			fmt.Printf("  1. Add DNS records (MX, SPF, DKIM, DMARC)\n")
			fmt.Printf("  2. Generate DKIM keys: mailstack dkim generate %s\n", domain)
			return nil
		},
	}
}

func domainDeleteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "delete <domain>",
		Short: "Delete a mail domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			db, err := database.Connect(cfg.Database)
			if err != nil {
				return err
			}
			defer db.Close()

			if err := db.DeleteDomain(domain); err != nil {
				return fmt.Errorf("failed to delete domain: %w", err)
			}

			fmt.Printf("✅ Domain %s deleted successfully\n", domain)
			return nil
		},
	}
}

func domainListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all mail domains",
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

			domains, err := db.ListDomains()
			if err != nil {
				return err
			}

			fmt.Println("🌐 Mail Domains:")
			for _, domain := range domains {
				fmt.Printf("  - %s (%d users)\n", domain.Name, domain.UserCount)
			}

			return nil
		},
	}
}
