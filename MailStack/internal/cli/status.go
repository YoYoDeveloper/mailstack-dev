package cli

import (
	"fmt"

	"github.com/mailstack/mailstack/internal/config"
	"github.com/mailstack/mailstack/internal/services"
	"github.com/spf13/cobra"
)

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check status of all services",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgFile)
			if err != nil {
				return err
			}

			mgr := services.NewManager(cfg)
			status, err := mgr.GetStatus()
			if err != nil {
				return err
			}

			fmt.Println("📊 MailStack Service Status:")

			for _, svc := range status {
				icon := "✅"
				if !svc.Running {
					icon = "❌"
				} else if !svc.Healthy {
					icon = "⚠️"
				}

				fmt.Printf("%s %-15s %s\n", icon, svc.Name, svc.Status)
			}

			return nil
		},
	}
}
