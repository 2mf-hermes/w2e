package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: peinspect FILE [FILE...]")
		os.Exit(2)
	}
	for _, path := range os.Args[1:] {
		if abs, err := filepath.Abs(path); err == nil {
			path = abs
		}
		st, err := os.Stat(path)
		if err != nil {
			fmt.Printf("%s\n  (error: %v)\n\n", path, err)
			continue
		}
		info, err := inspectPE(path)
		if err != nil {
			fmt.Printf("%s\n  size: %d bytes\n  PE parse error: %v\n\n", path, st.Size(), err)
			continue
		}
		fmt.Printf("%s\n", filepath.Base(path))
		fmt.Printf("  path            : %s\n", path)
		fmt.Printf("  size_bytes      : %d\n", st.Size())
		fmt.Printf("  size_human      : %s\n", humanSize(st.Size()))
		fmt.Printf("  format          : %s\n", info.Format)
		fmt.Printf("  machine         : %s\n", info.Machine)
		fmt.Printf("  optional_header : %s\n", info.OptionalHeader)
		fmt.Printf("  subsystem       : %s (0x%04X)\n", info.SubsystemName, info.Subsystem)
		fmt.Printf("  num_sections    : %d\n", info.NumSections)
		fmt.Printf("  entry_point_rva : 0x%08X\n", info.EntryPoint)
		if info.MajorLinker != 0 || info.MinorLinker != 0 {
			fmt.Printf("  linker_version  : %d.%d\n", info.MajorLinker, info.MinorLinker)
		}
		fmt.Println()
	}
}

type peInfo struct {
	Format         string
	Machine        string
	OptionalHeader string
	Subsystem      uint16
	SubsystemName  string
	NumSections    uint16
	EntryPoint     uint32
	MajorLinker    uint8
	MinorLinker    uint8
}

func inspectPE(path string) (*peInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var dos [64]byte
	if _, err := io.ReadFull(f, dos[:]); err != nil {
		return nil, fmt.Errorf("DOS header read: %w", err)
	}
	if binary.LittleEndian.Uint16(dos[0:2]) != 0x5A4D {
		return nil, fmt.Errorf("not MZ")
	}
	lfanew := int64(binary.LittleEndian.Uint32(dos[60:64]))
	if _, err := f.Seek(lfanew, io.SeekStart); err != nil {
		return nil, fmt.Errorf("seek PE: %w", err)
	}
	var sig [4]byte
	if _, err := io.ReadFull(f, sig[:]); err != nil {
		return nil, fmt.Errorf("PE sig: %w", err)
	}
	if !(sig[0] == 'P' && sig[1] == 'E' && sig[2] == 0 && sig[3] == 0) {
		return nil, fmt.Errorf("bad PE signature")
	}
	var coff [20]byte
	if _, err := io.ReadFull(f, coff[:]); err != nil {
		return nil, fmt.Errorf("COFF: %w", err)
	}
	machine := binary.LittleEndian.Uint16(coff[0:2])
	numSections := binary.LittleEndian.Uint16(coff[2:4])
	majorLinker := coff[4]
	minorLinker := coff[5]
	sizeOpt := binary.LittleEndian.Uint16(coff[16:18])

	info := &peInfo{
		Format:      "Windows Portable Executable (PE)",
		Machine:     machineName(machine),
		NumSections: numSections,
		MajorLinker: majorLinker,
		MinorLinker: minorLinker,
	}

	var opt [4]byte
	if _, err := io.ReadFull(f, opt[:]); err != nil {
		return nil, fmt.Errorf("opt magic: %w", err)
	}
	magic := binary.LittleEndian.Uint16(opt[0:2])
	switch magic {
	case 0x10b:
		info.OptionalHeader = "PE32 (32-bit)"
	case 0x20b:
		info.OptionalHeader = "PE32+ (64-bit)"
	default:
		info.OptionalHeader = fmt.Sprintf("unknown (0x%04X)", magic)
	}

	// Subsystem @ optional-header offset 0x44 (68). We already read 4 bytes
	// of the optional header, so skip (68 - 4) bytes then read 2.
	skip := 68 - 4
	buf := make([]byte, skip+2)
	if _, err := io.ReadFull(f, buf); err != nil {
		return nil, fmt.Errorf("optional header: %w", err)
	}
	sub := binary.LittleEndian.Uint16(buf[len(buf)-2:])
	info.Subsystem = sub
	info.SubsystemName = subsystemName(sub)

	// For PE32 vs PE32+, AddressOfEntryPoint is at optional-header offset 16.
	// We read 4 + (68-4) = 68 bytes of the optional header so far. To avoid a
	// second seek, approximate by seeking back to lfanew+4+20+16.
	if _, err := f.Seek(lfanew+4+20+16, io.SeekStart); err == nil {
		var ep [4]byte
		if _, err := io.ReadFull(f, ep[:]); err == nil {
			info.EntryPoint = binary.LittleEndian.Uint32(ep[:])
		}
	}
	_ = sizeOpt
	return info, nil
}

func machineName(m uint16) string {
	switch m {
	case 0x8664:
		return "AMD64 (x64, 0x8664)"
	case 0x014c:
		return "I386 (x86, 0x014c)"
	case 0xAA64:
		return "ARM64 (0xAA64)"
	case 0x01c0:
		return "ARM (0x01c0)"
	}
	return fmt.Sprintf("unknown (0x%04X)", m)
}

func subsystemName(s uint16) string {
	switch s {
	case 2:
		return "IMAGE_SUBSYSTEM_WINDOWS_GUI"
	case 3:
		return "IMAGE_SUBSYSTEM_WINDOWS_CUI (console)"
	case 9:
		return "EFI_APPLICATION"
	case 10:
		return "EFI_BOOT_SERVICE_DRIVER"
	case 11:
		return "EFI_RUNTIME_DRIVER"
	}
	return fmt.Sprintf("unknown (0x%04X)", s)
}

func humanSize(n int64) string {
	const u = 1024
	if n < u {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(u), 0
	for x := n / u; x >= u; x /= u {
		div *= u
		exp++
	}
	return fmt.Sprintf("%.2f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

var _ = strings.TrimSpace
