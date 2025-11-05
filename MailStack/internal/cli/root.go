package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
	verbose bool
)

// Execute runs the root command
func Execute(version, commit, date string) error {
	rootCmd := &cobra.Command{
		Use:   "mailstack",
		Short: "MailStack - Complete mail server installer and management",
		Long: `MailStack is a complete mail server solution that installs and manages
Postfix, Dovecot, Rspamd, Nginx, and other components on bare metal or VMs.`,
		Version: fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date),
	}

	// Global flags
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "mailstack.json", "config file")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(userCmd())
	rootCmd.AddCommand(domainCmd())
	rootCmd.AddCommand(aliasCmd())
	rootCmd.AddCommand(dkimCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(updateCmd())
	rootCmd.AddCommand(configCmd())

	return rootCmd.Execute()
}
