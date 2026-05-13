// Package cli 基于 cobra 构建的子命令集合。
package cli

import (
	"github.com/spf13/cobra"

	"github.com/florentzhu/macwechatRetracement/internal/wechat"
)

// 命令行通用选项
type globalOptions struct {
	app    string
	config string
}

// 默认远程配置地址
const defaultConfigURL = "https://raw.githubusercontent.com/florentzhu/macwechatRetracement/refs/heads/main/config.json"

// NewRootCommand 创建根命令，带 versions / patch 两个子命令。
func NewRootCommand() *cobra.Command {
	opts := &globalOptions{}

	root := &cobra.Command{
		Use:   "wechattweak",
		Short: "A command-line tool for tweaking WeChat.",
		Long:  "A command-line tool for tweaking WeChat (Go edition).",
		// 没有子命令时打印帮助
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().StringVarP(&opts.app, "app", "a",
		wechat.DefaultAppPath, "Path of WeChat.app")
	root.PersistentFlags().StringVarP(&opts.config, "config", "c",
		defaultConfigURL, "Local path or remote URL of config.json")

	root.AddCommand(newVersionsCommand(opts))
	root.AddCommand(newPatchCommand(opts))

	return root
}
