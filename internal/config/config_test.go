package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestArchCPU(t *testing.T) {
	if cpu, err := ArchARM64.CPU(); err != nil || cpu != CPUTypeARM64 {
		t.Fatalf("arm64 cpu got=0x%x err=%v", cpu, err)
	}
	if cpu, err := ArchX86_64.CPU(); err != nil || cpu != CPUTypeX86_64 {
		t.Fatalf("x86_64 cpu got=0x%x err=%v", cpu, err)
	}
	if _, err := Arch("ppc").CPU(); err == nil {
		t.Fatal("expected error for unsupported arch")
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "config.json")
	const data = `[
  {
    "version": "10000",
    "targets": [
      {
        "identifier": "revoke",
        "entries": [
          { "arch": "arm64", "addr": "100001000", "asm": "00008052C0035FD6" }
        ]
      }
    ]
  }
]`
	if err := os.WriteFile(p, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}

	cfgs, err := Load(p)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfgs) != 1 || cfgs[0].Version != "10000" {
		t.Fatalf("unexpected configs: %+v", cfgs)
	}
	c, ok := FindByVersion(cfgs, "10000")
	if !ok {
		t.Fatal("FindByVersion should find 10000")
	}
	if len(c.Targets) != 1 || c.Targets[0].Identifier != "revoke" {
		t.Fatalf("unexpected targets: %+v", c.Targets)
	}
	e := c.Targets[0].Entries[0]
	if e.Arch != ArchARM64 {
		t.Fatalf("arch=%s", e.Arch)
	}
	if e.Addr != 0x100001000 {
		t.Fatalf("addr=0x%x", e.Addr)
	}
	wantASM := []byte{0x00, 0x00, 0x80, 0x52, 0xC0, 0x03, 0x5F, 0xD6}
	if string(e.ASM) != string(wantASM) {
		t.Fatalf("asm=% X", e.ASM)
	}
}
