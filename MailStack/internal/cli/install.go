package cli

import (
	"fmt"
	"os"

	"github.com/mailstack/mailstack/internal/config"
	"github.com/mailstack/mailstack/internal/installer"
	"github.com/spf13/cobra"
)

func installCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install and configure the mail server",
		Long:  `Install all required components and configure the mail server based on the config file.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Check if running as root
			if os.Geteuid() != 0 {
				return fmt.Errorf("installation must be run as root")
			}

			// Load configuration
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Validate configuration
			if err := cfg.Validate(); err != nil {
				return fmt.Errorf("invalid configuration: %w", err)
			}

			// Create installer
			inst := installer.New(cfg, verbose)

			// Run installation
			fmt.Println("🚀 Starting MailStack installation...")
			if err := inst.Install(force); err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}

			fmt.Println("✅ MailStack installation completed successfully!")
			fmt.Printf("\n📧 Admin panel: https://%s/admin\n", cfg.Hostname)
			fmt.Printf("👤 Admin email: %s\n", cfg.Admin.Email)
			fmt.Println("\n🔒 Please change the admin password after first login!")

			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "force reinstallation even if already installed")

	return cmd
}
