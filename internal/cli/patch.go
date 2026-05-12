package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/sunnyyoung/wechattweak/internal/config"
	"github.com/sunnyyoung/wechattweak/internal/patcher"
	"github.com/sunnyyoung/wechattweak/internal/wechat"
)

// ErrUnsupportedVersion 找不到当前 WeChat 版本对应的 patch 配置时返回。
var ErrUnsupportedVersion = errors.New("unsupported WeChat version")

func newPatchCommand(opts *globalOptions) *cobra.Command {
	return &cobra.Command{
		Use:   "patch",
		Short: "Patch WeChat.app",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := wechat.EnsureAppExists(opts.app); err != nil {
				return err
			}

			fmt.Println("------ Version ------")
			version, err := wechat.ReadVersion(opts.app)
			if err != nil {
				return fmt.Errorf("read WeChat version: %w", err)
			}
			fmt.Printf("WeChat version: %s\n", version)

			fmt.Println("------ Config ------")
			cfgs, err := config.Load(opts.config)
			if err != nil {
				return err
			}
			cfg, ok := config.FindByVersion(cfgs, version)
			if !ok {
				return fmt.Errorf("%w: %s", ErrUnsupportedVersion, version)
			}
			fmt.Printf("Matched config: version=%s targets=%d\n",
				cfg.Version, len(cfg.Targets))
			for _, t := range cfg.Targets {
				fmt.Printf("  - %s (%d entries)\n", t.Identifier, len(t.Entries))
			}

			fmt.Println("------ Patch ------")
			if err := patcher.Patch(wechat.BinaryPath(opts.app), cfg); err != nil {
				return fmt.Errorf("patch failed: %w", err)
			}
			fmt.Println("Done!")

			fmt.Println("------ Resign ------")
			if err := wechat.Resign(opts.app); err != nil {
				return fmt.Errorf("resign failed: %w", err)
			}
			fmt.Println("Done!")
			return nil
		},
	}
}
