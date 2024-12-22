package main

import (
	"context"
	"fmt"
	"os"

	"github.com/adrg/xdg"
	"github.com/pkg/errors"
	wslopenproxy "github.com/qnighy/wsl-open-proxy"
	"github.com/spf13/cobra"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:     "setup-wsl-open",
		Version: wslopenproxy.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return errors.New("too many arguments")
			}
			cmd.SilenceUsage = true
			return run(cmd.Context())
		},
	}

	err := rootCmd.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	_ = ctx
	fmt.Printf("config dir = %s\n", xdg.ConfigHome)
	fmt.Printf("data dir = %s\n", xdg.DataHome)
	return nil
}
