// wechattweak 是 WeChatTweak 的 Go 版本入口。
package main

import (
	"fmt"
	"os"

	"github.com/florentzhu/macwechatRetracement/internal/cli"
)

func main() {
	if err := cli.NewRootCommand().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
