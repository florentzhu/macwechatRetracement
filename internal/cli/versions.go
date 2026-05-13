package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/florentzhu/macwechatRetracement/internal/config"
	"github.com/florentzhu/macwechatRetracement/internal/wechat"
)

func newVersionsCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "versions",
		Short: "List all supported WeChat versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("------ Current version ------")
			if err := wechat.EnsureAppExists(opts.app); err != nil {
				fmt.Println("unknown")
			} else {
				v, err := wechat.ReadVersion(opts.app)
				if err != nil {
					fmt.Println("unknown")
				} else {
					fmt.Println(v)
				}
			}

			fmt.Println("------ Supported versions ------")
			cfgs, err := config.Load(opts.config)
			if err != nil {
				return err
			}
			for _, c := range cfgs {
				fmt.Println(c.Version)
			}
			return nil
		},
	}
}
