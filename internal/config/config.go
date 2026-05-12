// Package config 负责加载 / 解析 WeChatTweak 的 config.json，
// 描述每个 WeChat 版本对应的需要 patch 的二进制位置与字节。
package config

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Arch 代表 CPU 架构。与 Mach-O 中的 cputype 对应。
type Arch string

const (
	ArchARM64  Arch = "arm64"
	ArchX86_64 Arch = "x86_64"
)

// CPU 类型常量，与 <mach/machine.h> 中的定义保持一致。
const (
	cpuArchABI64 uint32 = 0x01000000
	cpuTypeX86   uint32 = 7
	cpuTypeARM   uint32 = 12

	CPUTypeX86_64 uint32 = cpuTypeX86 | cpuArchABI64 // 0x01000007
	CPUTypeARM64  uint32 = cpuTypeARM | cpuArchABI64 // 0x0100000C
)

// CPU 返回该架构对应的 Mach-O cputype 值。
func (a Arch) CPU() (uint32, error) {
	switch a {
	case ArchARM64:
		return CPUTypeARM64, nil
	case ArchX86_64:
		return CPUTypeX86_64, nil
	default:
		return 0, fmt.Errorf("unsupported arch: %s", a)
	}
}

// Entry 是一条 patch 记录：在指定架构的某个虚拟地址处写入若干字节。
type Entry struct {
	Arch Arch   `json:"arch"`
	Addr uint64 `json:"-"` // 实际值，由 AddrHex 解码得到
	ASM  []byte `json:"-"` // 实际值，由 ASMHex 解码得到

	AddrHex string `json:"addr"`
	ASMHex  string `json:"asm"`
}

// UnmarshalJSON 自定义解码：将 hex 字符串转成数值与字节序列。
func (e *Entry) UnmarshalJSON(data []byte) error {
	type raw struct {
		Arch Arch   `json:"arch"`
		Addr string `json:"addr"`
		ASM  string `json:"asm"`
	}
	var r raw
	if err := json.Unmarshal(data, &r); err != nil {
		return err
	}

	addr, err := strconv.ParseUint(strings.TrimPrefix(r.Addr, "0x"), 16, 64)
	if err != nil {
		return fmt.Errorf("invalid Entry.addr %q: %w", r.Addr, err)
	}

	asm, err := hex.DecodeString(r.ASM)
	if err != nil {
		return fmt.Errorf("invalid Entry.asm %q: %w", r.ASM, err)
	}

	e.Arch = r.Arch
	e.Addr = addr
	e.ASM = asm
	e.AddrHex = r.Addr
	e.ASMHex = r.ASM
	return nil
}

// Target 表示对应某个功能（identifier）的 patch 集合。
type Target struct {
	Identifier string  `json:"identifier"`
	Entries    []Entry `json:"entries"`
}

// Config 表示某个 WeChat 版本对应的所有 patch 信息。
type Config struct {
	Version string   `json:"version"`
	Targets []Target `json:"targets"`
}

// Load 从本地路径或者 http(s) URL 加载配置。
func Load(source string) ([]Config, error) {
	if source == "" {
		return nil, errors.New("empty config source")
	}

	var data []byte
	var err error

	if isHTTPURL(source) {
		data, err = fetchHTTP(source)
	} else {
		data, err = os.ReadFile(source)
	}
	if err != nil {
		return nil, fmt.Errorf("read config %q: %w", source, err)
	}

	var configs []Config
	if err := json.Unmarshal(data, &configs); err != nil {
		return nil, fmt.Errorf("parse config %q: %w", source, err)
	}
	return configs, nil
}

// FindByVersion 在配置列表中查找指定版本号的配置。
func FindByVersion(configs []Config, version string) (*Config, bool) {
	for i := range configs {
		if configs[i].Version == version {
			return &configs[i], true
		}
	}
	return nil, false
}

func isHTTPURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}

func fetchHTTP(url string) ([]byte, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return nil, fmt.Errorf("http %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
