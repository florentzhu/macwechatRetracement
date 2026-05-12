// Package wechat 提供与 WeChat.app 相关的辅助：定位主二进制、读取版本号、重签名。
package wechat

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DefaultAppPath 是 macOS 上 WeChat.app 默认安装路径。
const DefaultAppPath = "/Applications/WeChat.app"

// EnsureAppExists 校验给定路径是否是一个存在的 .app bundle。
func EnsureAppExists(appPath string) error {
	info, err := os.Stat(appPath)
	if err != nil {
		return fmt.Errorf("invalid app path %q: %w", appPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("invalid app path %q: not a directory", appPath)
	}
	return nil
}

// BinaryPath 返回 WeChat.app 内部的主可执行文件路径。
func BinaryPath(appPath string) string {
	return filepath.Join(appPath, "Contents", "MacOS", "WeChat")
}

// InfoPlistPath 返回 Info.plist 路径。
func InfoPlistPath(appPath string) string {
	return filepath.Join(appPath, "Contents", "Info.plist")
}

// ReadVersion 通过 `defaults read` 读取 CFBundleVersion。
func ReadVersion(appPath string) (string, error) {
	plist := InfoPlistPath(appPath)
	if _, err := os.Stat(plist); err != nil {
		return "", fmt.Errorf("Info.plist not found: %w", err)
	}

	// `defaults read` 要求路径不带 .plist 扩展名
	plistKey := strings.TrimSuffix(plist, ".plist")
	cmd := exec.Command("defaults", "read", plistKey, "CFBundleVersion")
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("defaults read CFBundleVersion: %w", err)
	}
	version := strings.TrimSpace(string(out))
	if version == "" {
		return "", errors.New("empty CFBundleVersion")
	}
	return version, nil
}

// Resign 删除签名 -> ad-hoc 重签 -> 清除 quarantine 等扩展属性。
// 步骤与 Swift 版本完全一致。
func Resign(appPath string) error {
	steps := [][]string{
		{"codesign", "--remove-signature", appPath},
		{"codesign", "--force", "--deep", "--sign", "-", appPath},
		{"xattr", "-cr", appPath},
	}
	for _, args := range steps {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("run %s: %w", strings.Join(args, " "), err)
		}
	}
	return nil
}
