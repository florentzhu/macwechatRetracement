# WeChatTweak (Go Edition)

A command-line tool for tweaking WeChat on macOS, rewritten in **Go**.

> 这是 [WeChatTweak](https://github.com/sunnyyoung/WeChatTweak) 的 Go 重构版本，行为与原 Swift 版保持一致：定位 WeChat.app 主二进制，按 `config.json` 中的 VA → 文件偏移定位，写入指定字节，最后 ad-hoc 重签名。

## 功能

- 阻止消息撤回
- 阻止自动更新
- 客户端多开

## 安装

仅支持 macOS。一行命令安装到 `/usr/local/bin/wechattweak`：

```bash
curl -fsSL https://raw.githubusercontent.com/florentzhu/macwechatRetracement/main/install.sh | sudo bash
```

可选环境变量：

- `PREFIX` 安装目录，默认 `/usr/local/bin`
- `VERSION` 指定版本（如 `v0.1.0`），默认 `latest`

例如安装到 `~/bin`、指定版本：

```bash
curl -fsSL https://raw.githubusercontent.com/florentzhu/macwechatRetracement/main/install.sh \
  | PREFIX="$HOME/bin" VERSION=v0.1.0 bash
```

验证：

```bash
wechattweak versions
```

## 使用

```bash
# 查看支持的 WeChat 版本
wechattweak versions

# 对默认路径 /Applications/WeChat.app 执行 patch
wechattweak patch

# 自定义 WeChat.app 路径与 config 来源（本地或 URL 均可）
wechattweak patch -a /Applications/WeChat.app -c ./config.json
wechattweak patch -c https://raw.githubusercontent.com/florentzhu/macwechatRetracement/refs/heads/main/config.json
```

执行 `patch` 时建议先退出 WeChat。如果系统提示「已损坏」或「无法验证开发者」，工具最后会自动 `codesign --force --deep --sign -` 并 `xattr -cr` 清理隔离属性。

## 项目结构

```
.
├── cmd/wechattweak/        # 入口 main 包
├── internal/
│   ├── cli/                # cobra 子命令: versions / patch
│   ├── config/             # config.json 加载与解析
│   ├── patcher/            # Mach-O 解析与 VA 写入
│   └── wechat/             # WeChat.app 定位、版本、重签名
├── config.json             # 各 WeChat 版本的 patch 数据
├── install.sh              # 一键安装脚本
└── go.mod
```

## License

[AGPL-3.0](LICENSE)
