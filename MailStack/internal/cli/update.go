package cli

import (
	"fmt"

	"github.com/mailstack/mailstack/internal/config"
	"github.com/mailstack/mailstack/internal/installer"
	"github.com/spf13/cobra"
)

func updateCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "update",
		Short: "Update MailStack components",
		Long:  `Update all mail server components and run database migrations.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			inst := installer.New(cfg, verbose)

			fmt.Println("🔄 Updating MailStack components...")
			if err := inst.Update(); err != nil {
				return fmt.Errorf("update failed: %w", err)
			}

			fmt.Println("✅ MailStack updated successfully!")
			return nil
		},
	}
}
