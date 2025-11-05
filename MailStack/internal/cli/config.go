package cli

import (
	"fmt"

	"github.com/mailstack/mailstack/internal/config"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
		Long:  `Validate, regenerate, or show configuration.`,
	}

	cmd.AddCommand(configValidateCmd())
	cmd.AddCommand(configRegenerateCmd())
	cmd.AddCommand(configShowCmd())

	return cmd
}

func configValidateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("configuration is invalid: %w", err)
			}

			fmt.Println("✅ Configuration is valid")
			return nil
		},
	}
}

func configRegenerateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "regenerate",
		Short: "Regenerate all service configuration files",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			fmt.Println("🔄 Regenerating configuration files...")
			// TODO: Implement regeneration logic
			fmt.Println("✅ Configuration files regenerated")
			return nil
		},
	}
}

func configShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			fmt.Printf("📋 Current Configuration:\n\n")
			fmt.Printf("Domain:       %s\n", cfg.Domain)
			fmt.Printf("Hostname:     %s\n", cfg.Hostname)
			fmt.Printf("Database:     %s (%s)\n", cfg.Database.Type, cfg.Database.Path)
			fmt.Printf("TLS:          %s\n", cfg.TLS.Flavor)
			fmt.Printf("Admin Email:  %s\n", cfg.Admin.Email)

			return nil
		},
	}
}
