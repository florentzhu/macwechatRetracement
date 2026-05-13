package patcher

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/florentzhu/macwechatRetracement/internal/config"
)

// 构造一个最小可用的 thin Mach-O 64 文件：
//
//	header(32) + LC_SEGMENT_64(72: 8 head + 64 body) + payload(0x100 字节)
//
// 段 vmaddr=0x100000000，vmsize=0x1000，fileoff=0x68（紧跟在 header+lc 之后），filesize=0x100
// 所以 VA 0x100000010 对应文件偏移 0x68 + 0x10 = 0x78
func writeFakeMacho(t *testing.T, path string) {
	t.Helper()

	const (
		headerSize = 32
		lcHeadSize = 8
		segBody    = 64
		payload    = 0x100
		vmaddr     = uint64(0x100000000)
		vmsize     = uint64(0x1000)
		fileoff    = uint64(headerSize + lcHeadSize + segBody)
	)

	buf := make([]byte, headerSize+lcHeadSize+segBody+payload)

	// mach_header_64
	binary.LittleEndian.PutUint32(buf[0:4], machoMagic64)        // magic
	binary.LittleEndian.PutUint32(buf[4:8], config.CPUTypeARM64) // cputype
	binary.LittleEndian.PutUint32(buf[8:12], 0)                  // cpusubtype
	binary.LittleEndian.PutUint32(buf[12:16], 2)                 // filetype MH_EXECUTE
	binary.LittleEndian.PutUint32(buf[16:20], 1)                 // ncmds
	binary.LittleEndian.PutUint32(buf[20:24], lcHeadSize+segBody)
	binary.LittleEndian.PutUint32(buf[24:28], 0)
	binary.LittleEndian.PutUint32(buf[28:32], 0)

	// LC_SEGMENT_64
	binary.LittleEndian.PutUint32(buf[32:36], lcSegment64)
	binary.LittleEndian.PutUint32(buf[36:40], lcHeadSize+segBody)

	// segname[16] -> 偏移 40..56，留全 0
	binary.LittleEndian.PutUint64(buf[56:64], vmaddr)
	binary.LittleEndian.PutUint64(buf[64:72], vmsize)
	binary.LittleEndian.PutUint64(buf[72:80], fileoff)
	binary.LittleEndian.PutUint64(buf[80:88], payload)
	// 其余 (maxprot, initprot, nsects, flags) = 0

	if err := os.WriteFile(path, buf, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestPatchThin(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake")
	writeFakeMacho(t, bin)

	want := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	cfg := &config.Config{
		Version: "test",
		Targets: []config.Target{{
			Identifier: "demo",
			Entries: []config.Entry{{
				Arch: config.ArchARM64,
				Addr: 0x100000010,
				ASM:  want,
			}},
		}},
	}

	if err := Patch(bin, cfg); err != nil {
		t.Fatalf("patch: %v", err)
	}

	got, err := os.ReadFile(bin)
	if err != nil {
		t.Fatal(err)
	}
	const wantOff = 32 + 8 + 64 + 0x10
	if string(got[wantOff:wantOff+len(want)]) != string(want) {
		t.Fatalf("patch bytes mismatch at 0x%x: % X", wantOff, got[wantOff:wantOff+len(want)])
	}
}

func TestPatchVANotFound(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "fake")
	writeFakeMacho(t, bin)

	cfg := &config.Config{
		Version: "test",
		Targets: []config.Target{{
			Identifier: "demo",
			Entries: []config.Entry{{
				Arch: config.ArchARM64,
				Addr: 0x200000000, // 段外
				ASM:  []byte{0x00},
			}},
		}},
	}

	if err := Patch(bin, cfg); err == nil {
		t.Fatal("expected ErrVANotFound")
	}
}
