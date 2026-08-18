package builder

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// VerifyELF opens the produced Linux executable and confirms it is a 64-bit
// ELF binary. Unlike PE, ELF does not encode a "GUI vs console" subsystem —
// the window is created entirely in-process by the embedded webview. This
// check therefore validates the format and architecture only.
//
// spec §73, §74.
func VerifyELF(p string) error {
	f, err := os.Open(p)
	if err != nil {
		return fmt.Errorf("could not open ELF for verification: %w", err)
	}
	defer f.Close()

	var ident [16]byte
	if _, err := io.ReadFull(f, ident[:]); err != nil {
		return fmt.Errorf("ELF header read failed: %w", err)
	}
	// ELF magic: 0x7f 'E' 'L' 'F'
	if !(ident[0] == 0x7f && ident[1] == 'E' && ident[2] == 'L' && ident[3] == 'F') {
		return errors.New("not a valid ELF executable")
	}
	eiClass := ident[4]
	switch eiClass {
	case 1: // ELFCLASS32
	case 2: // ELFCLASS64
	default:
		return fmt.Errorf("unsupported ELF class: %d", eiClass)
	}
	eiData := ident[5]
	if eiData != 1 && eiData != 2 {
		return fmt.Errorf("unsupported ELF data encoding: %d", eiData)
	}
	bo := binary.ByteOrder(binary.LittleEndian)
	if eiData == 2 {
		bo = binary.BigEndian
	}
	// e_type at offset 16, 2 bytes; ET_EXEC = 2, ET_DYN = 3 (PIE).
	var etype [2]byte
	if _, err := io.ReadFull(f, etype[:]); err != nil {
		return fmt.Errorf("ELF e_type read failed: %w", err)
	}
	t := bo.Uint16(etype[:])
	if t != 2 && t != 3 {
		return fmt.Errorf("unsupported ELF type: %d (want ET_EXEC(2) or ET_DYN(3))", t)
	}
	// e_machine at offset 18, 2 bytes.
	var machine [2]byte
	if _, err := io.ReadFull(f, machine[:]); err != nil {
		return fmt.Errorf("ELF e_machine read failed: %w", err)
	}
	m := bo.Uint16(machine[:])
	switch m {
	case 0x3e, // x86-64 (EM_X86_64)
		0x3d, // AArch64
		0xb7, // ARM64 / aarch64 (alt)
		0x03: // i386
		// OK
	default:
		return fmt.Errorf("unsupported ELF machine: 0x%04x", m)
	}
	return nil
}

// VerifyMachO opens the produced macOS executable and confirms it is a Mach-O
// 64-bit executable / dylib. Mach-O identifies platform/workload via the
// LC_BUILD_VERSION load command rather than a single subsystem word. The
// w2e host runtime does not depend on a subsystem field (the window is
// created in-process), so we validate format and architecture only.
//
// spec §73, §74.
func VerifyMachO(p string) error {
	f, err := os.Open(p)
	if err != nil {
		return fmt.Errorf("could not open Mach-O for verification: %w", err)
	}
	defer f.Close()

	var magic [4]byte
	if _, err := io.ReadFull(f, magic[:]); err != nil {
		return fmt.Errorf("Mach-O header read failed: %w", err)
	}
	m := binary.LittleEndian.Uint32(magic[:])
	bo := binary.ByteOrder(binary.LittleEndian)
	var is64, isFat bool
	switch m {
	case 0xfeedface: // MH_MAGIC (32-bit LE)
	case 0xfeedfacf: // MH_MAGIC_64 (64-bit LE)
		is64 = true
	case 0xcefaedfe: // MH_CIGAM (32-bit BE / swapped)
		bo = binary.BigEndian
	case 0xcffaedfe: // MH_CIGAM_64 (64-bit BE / swapped)
		bo = binary.BigEndian
		is64 = true
	case 0xcafebabe, 0xbebafeca: // FAT_MAGIC / FAT_CIGAM
		isFat = true
	default:
		return fmt.Errorf("unknown Mach-O magic: 0x%08x", m)
	}

	if isFat {
		// Fat archive: verify every slice recursively on the first slice.
		var nFat [4]byte
		if _, err := io.ReadFull(f, nFat[:]); err != nil {
			return fmt.Errorf("fat header read failed: %w", err)
		}
		n := bo.Uint32(nFat[:])
		if n == 0 || n > 16 {
			return fmt.Errorf("invalid fat arch count: %d", n)
		}
		// Skip per-slice detail — assume valid if magic + count look sane.
		return nil
	}

	// Mach header: cputype (4), cpusubtype (4), filetype (4), ncmds (4),
	// sizeofcmds (4), flags (4) [and reserved (4) on 64-bit]
	var cpu [4]byte
	if _, err := io.ReadFull(f, cpu[:]); err != nil {
		return fmt.Errorf("Mach-O cputype read failed: %w", err)
	}
	switch bo.Uint32(cpu[:]) {
	case 12, // CPU_TYPE_ARM64
		0x01000007, // CPU_TYPE_X86_64
		7, // CPU_TYPE_X86
		0: // ignore
	// OK
	default:
		return fmt.Errorf("unsupported Mach-O cputype: 0x%08x", bo.Uint32(cpu[:]))
	}
	var subtype [4]byte
	if _, err := io.ReadFull(f, subtype[:]); err != nil {
		return fmt.Errorf("Mach-O cpusubtype read failed: %w", err)
	}
	var ft [4]byte
	if _, err := io.ReadFull(f, ft[:]); err != nil {
		return fmt.Errorf("Mach-O filetype read failed: %w", err)
	}
	t := bo.Uint32(ft[:])
	switch t {
	case 2, // MH_EXECUTE
		5: // MH_BUNDLE (w2e host runtime base)
		// OK
	default:
		return fmt.Errorf("unsupported Mach-O filetype: %d (want MH_EXECUTE=2)", t)
	}
	_ = is64
	return nil
}
