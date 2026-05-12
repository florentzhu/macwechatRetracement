// Package patcher 负责直接对 Mach-O 二进制按虚拟地址（VA）打补丁。
// 同时支持 thin Mach-O（MH_MAGIC_64）和 fat Mach-O（FAT_MAGIC / FAT_CIGAM）。
package patcher

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/sunnyyoung/wechattweak/internal/config"
)

// Mach-O 与 Fat 头部 magic 常量。
const (
	machoMagic64 uint32 = 0xFEEDFACF // MH_MAGIC_64
	fatMagic     uint32 = 0xCAFEBABE // FAT_MAGIC
	fatCigam     uint32 = 0xBEBAFECA // FAT_CIGAM (byte swapped)

	lcSegment64 uint32 = 0x19 // LC_SEGMENT_64
)

// 错误类型
var (
	ErrInvalidFile     = errors.New("invalid file")
	ErrNot64BitMachO   = errors.New("not a 64-bit Mach-O")
	ErrNoArchMatched   = errors.New("no arch matched")
	ErrVANotFound      = errors.New("virtual address not found in any segment")
	ErrEmptyEntries    = errors.New("config has no entries")
)

// Patch 对 binaryPath 指向的 Mach-O 文件按 cfg 中的 entries 执行原地替换。
func Patch(binaryPath string, cfg *config.Config) error {
	if _, err := os.Stat(binaryPath); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFile, err)
	}

	var entries []config.Entry
	for _, t := range cfg.Targets {
		entries = append(entries, t.Entries...)
	}
	if len(entries) == 0 {
		return ErrEmptyEntries
	}

	f, err := os.OpenFile(binaryPath, os.O_RDWR, 0)
	if err != nil {
		return fmt.Errorf("open binary: %w", err)
	}
	defer f.Close()

	// 读取头 4 字节判断 fat / thin
	var magicBuf [4]byte
	if _, err := io.ReadFull(f, magicBuf[:]); err != nil {
		return fmt.Errorf("%w: read magic: %v", ErrInvalidFile, err)
	}
	magicBE := binary.BigEndian.Uint32(magicBuf[:])

	patched := 0
	switch magicBE {
	case fatMagic, fatCigam:
		isSwapped := magicBE == fatCigam
		n, err := patchFat(f, entries, isSwapped)
		if err != nil {
			return err
		}
		patched += n
	default:
		// 当成 thin Mach-O 处理：从头按小端解析
		n, err := patchThin(f, entries)
		if err != nil {
			return err
		}
		patched += n
	}

	if patched <= 0 {
		return ErrNoArchMatched
	}
	return nil
}

// patchFat 处理 fat Mach-O。文件指针此时位于 magic 之后（offset 4）。
func patchFat(f *os.File, entries []config.Entry, isSwapped bool) (int, error) {
	// 读取 nfat_arch
	var nfatBuf [4]byte
	if _, err := io.ReadFull(f, nfatBuf[:]); err != nil {
		return 0, fmt.Errorf("%w: read nfat: %v", ErrInvalidFile, err)
	}
	var nfat uint32
	if isSwapped {
		nfat = binary.LittleEndian.Uint32(nfatBuf[:])
	} else {
		nfat = binary.BigEndian.Uint32(nfatBuf[:])
	}

	// fat_arch: cputype(4) cpusubtype(4) offset(4) size(4) align(4) -> 20 bytes，big-endian
	type archEntry struct {
		cpuType uint32
		offset  uint32
	}
	archs := make([]archEntry, 0, nfat)
	for i := uint32(0); i < nfat; i++ {
		var raw [20]byte
		if _, err := io.ReadFull(f, raw[:]); err != nil {
			return 0, fmt.Errorf("%w: read fat_arch: %v", ErrInvalidFile, err)
		}
		var cpu, off uint32
		if isSwapped {
			cpu = binary.LittleEndian.Uint32(raw[0:4])
			off = binary.LittleEndian.Uint32(raw[8:12])
		} else {
			cpu = binary.BigEndian.Uint32(raw[0:4])
			off = binary.BigEndian.Uint32(raw[8:12])
		}
		archs = append(archs, archEntry{cpuType: cpu, offset: off})
	}

	patched := 0
	for _, a := range archs {
		for _, ent := range entries {
			cpu, err := ent.Arch.CPU()
			if err != nil || cpu != a.cpuType {
				continue
			}
			if err := patchOneSlice(f, uint64(a.offset), ent); err != nil {
				return patched, err
			}
			patched++
		}
	}
	return patched, nil
}

// patchThin 处理 thin Mach-O。文件指针位于 magic 之后，但我们会先回到 0 从头读 header。
func patchThin(f *os.File, entries []config.Entry) (int, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return 0, fmt.Errorf("seek thin start: %w", err)
	}

	var hdr [32]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return 0, fmt.Errorf("%w: read thin header: %v", ErrInvalidFile, err)
	}
	magic := binary.LittleEndian.Uint32(hdr[0:4])
	if magic != machoMagic64 {
		return 0, fmt.Errorf("%w: magic=0x%x", ErrNot64BitMachO, magic)
	}
	cpuType := binary.LittleEndian.Uint32(hdr[4:8])

	patched := 0
	for _, ent := range entries {
		cpu, err := ent.Arch.CPU()
		if err != nil || cpu != cpuType {
			continue
		}
		if err := patchOneSlice(f, 0, ent); err != nil {
			return patched, err
		}
		patched++
	}
	if patched == 0 {
		return 0, ErrNoArchMatched
	}
	return patched, nil
}

// patchOneSlice 在某个 slice 内（fat 中的某个 arch 或 thin 整个文件，sliceOffset=0）
// 找到包含 targetVA 的 LC_SEGMENT_64，把 ent.ASM 写入对应的文件偏移。
func patchOneSlice(f *os.File, sliceOffset uint64, ent config.Entry) error {
	if _, err := f.Seek(int64(sliceOffset), io.SeekStart); err != nil {
		return fmt.Errorf("seek slice: %w", err)
	}

	var hdr [32]byte
	if _, err := io.ReadFull(f, hdr[:]); err != nil {
		return fmt.Errorf("%w: read slice header: %v", ErrInvalidFile, err)
	}

	magic := binary.LittleEndian.Uint32(hdr[0:4])
	if magic != machoMagic64 {
		return fmt.Errorf("%w: slice magic=0x%x", ErrNot64BitMachO, magic)
	}
	ncmds := binary.LittleEndian.Uint32(hdr[16:20])

	lcOffset := sliceOffset + 32

	for i := uint32(0); i < ncmds; i++ {
		if _, err := f.Seek(int64(lcOffset), io.SeekStart); err != nil {
			return fmt.Errorf("seek lc: %w", err)
		}

		var lcHead [8]byte
		if _, err := io.ReadFull(f, lcHead[:]); err != nil {
			return fmt.Errorf("%w: read lc head: %v", ErrInvalidFile, err)
		}
		cmd := binary.LittleEndian.Uint32(lcHead[0:4])
		cmdsize := binary.LittleEndian.Uint32(lcHead[4:8])

		if cmd == lcSegment64 {
			// segment_command_64 紧跟在 cmd/cmdsize 后面，剩余 64 字节包含
			// segname(16) vmaddr(8) vmsize(8) fileoff(8) filesize(8) ...
			var seg [64]byte
			if _, err := io.ReadFull(f, seg[:]); err != nil {
				return fmt.Errorf("%w: read segment: %v", ErrInvalidFile, err)
			}
			vmaddr := binary.LittleEndian.Uint64(seg[16:24])
			vmsize := binary.LittleEndian.Uint64(seg[24:32])
			fileoff := binary.LittleEndian.Uint64(seg[32:40])

			if vmaddr <= ent.Addr && ent.Addr < vmaddr+vmsize {
				fileOffset := sliceOffset + fileoff + (ent.Addr - vmaddr)

				fmt.Printf("[%s] vmaddr=0x%x, fileoff=0x%x, sliceoff=0x%x\n",
					ent.Arch, vmaddr, fileoff, sliceOffset)
				fmt.Printf("[%s] patch VA=0x%x, fileoff=0x%x, bytes=% X\n",
					ent.Arch, ent.Addr, fileOffset, ent.ASM)

				if _, err := f.Seek(int64(fileOffset), io.SeekStart); err != nil {
					return fmt.Errorf("seek patch offset: %w", err)
				}
				if _, err := f.Write(ent.ASM); err != nil {
					return fmt.Errorf("write patch: %w", err)
				}
				return nil
			}
		}

		lcOffset += uint64(cmdsize)
	}

	return fmt.Errorf("%w: arch=%s va=0x%x", ErrVANotFound, ent.Arch, ent.Addr)
}
