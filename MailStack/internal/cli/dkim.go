package cli

import (
	"fmt"

	"github.com/mailstack/mailstack/internal/config"
	"github.com/mailstack/mailstack/internal/dkim"
	"github.com/spf13/cobra"
)

func dkimCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dkim",
		Short: "Manage DKIM keys",
		Long:  `Generate and manage DKIM keys for domains.`,
	}

	cmd.AddCommand(dkimGenerateCmd())
	cmd.AddCommand(dkimShowCmd())

	return cmd
}

func dkimGenerateCmd() *cobra.Command {
	var selector string
	var bits int

	cmd := &cobra.Command{
		Use:   "generate <domain>",
		Short: "Generate DKIM key for domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			keyPath, dnsRecord, err := dkim.Generate(domain, selector, bits, cfg.DKIMPath)
			if err != nil {
				return fmt.Errorf("failed to generate DKIM key: %w", err)
			}

			fmt.Printf("✅ DKIM key generated successfully\n")
			fmt.Printf("📁 Key saved to: %s\n\n", keyPath)
			fmt.Println("📝 Add this TXT record to your DNS:")
			fmt.Printf("   %s._domainkey.%s IN TXT \"%s\"\n", selector, domain, dnsRecord)

			return nil
		},
	}

	cmd.Flags().StringVarP(&selector, "selector", "s", "dkim", "DKIM selector")
	cmd.Flags().IntVarP(&bits, "bits", "b", 2048, "RSA key size (1024, 2048, or 4096)")

	return cmd
}

func dkimShowCmd() *cobra.Command {
	var selector string

	cmd := &cobra.Command{
		Use:   "show <domain>",
		Short: "Show DKIM DNS record for domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			domain := args[0]

			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			dnsRecord, err := dkim.GetDNSRecord(domain, selector, cfg.DKIMPath)
			if err != nil {
				return fmt.Errorf("failed to read DKIM key: %w", err)
			}

			fmt.Println("📝 DKIM DNS TXT record:")
			fmt.Printf("   %s._domainkey.%s IN TXT \"%s\"\n", selector, domain, dnsRecord)

			return nil
		},
	}

	cmd.Flags().StringVarP(&selector, "selector", "s", "dkim", "DKIM selector")

	return cmd
}
