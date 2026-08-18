package builder

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
)

// Windows PE subsystem constants used for verification.
const (
	imageSubsystemWindowsGUI = 2
	imageSubsystemWindowsCUI = 3

	mz = 0x5A4D
)

// VerifyPE opens the produced EXE and confirms it is a Windows PE with the
// GUI subsystem (so no console window will open). Returns an error with a
// stale errcode-friendly message on failure; callers wrap with verify.
//
// spec §73, §74.
func VerifyPE(p string) error {
	f, err := os.Open(p)
	if err != nil {
		return fmt.Errorf("could not open EXE for verification: %w", err)
	}
	defer f.Close()

	// DOS header (MZ)
	var dos [64]byte
	if _, err := io.ReadFull(f, dos[:]); err != nil {
		return fmt.Errorf("DOS header read failed: %w", err)
	}
	if binary.LittleEndian.Uint16(dos[0:2]) != mz {
		return errors.New("not a valid MZ/PE executable")
	}
	// e_lfanew at offset 0x3C points to the PE header.
	lfanew := int64(binary.LittleEndian.Uint32(dos[60:64]))
	if _, err := f.Seek(lfanew, io.SeekStart); err != nil {
		return fmt.Errorf("seek to PE header failed: %w", err)
	}
	var sig [4]byte
	if _, err := io.ReadFull(f, sig[:]); err != nil {
		return fmt.Errorf("PE signature read failed: %w", err)
	}
	// PE\0\0
	if !(sig[0] == 'P' && sig[1] == 'E' && sig[2] == 0 && sig[3] == 0) {
		return errors.New("invalid PE signature")
	}

	// COFF header is 20 bytes after "PE\0\0".
	var coff [20]byte
	if _, err := io.ReadFull(f, coff[:]); err != nil {
		return fmt.Errorf("COFF header read failed: %w", err)
	}
	machine := binary.LittleEndian.Uint16(coff[0:2])
	if !validMachine(machine) {
		return fmt.Errorf("unsupported PE machine: 0x%04x (expected amd64/arm64/i386)", machine)
	}
	numSections := binary.LittleEndian.Uint16(coff[2:4])
	sizeOfOptionalHeader := binary.LittleEndian.Uint16(coff[16:18])
	if sizeOfOptionalHeader < 68 { // optional header min size on PE32/PE32+
		return errors.New("PE optional header too small")
	}

	// Optional header begins right after the COFF header.
	var opt [4]byte
	if _, err := io.ReadFull(f, opt[:]); err != nil {
		return fmt.Errorf("optional header magic read failed: %w", err)
	}
	pe32Plus := false
	switch binary.LittleEndian.Uint16(opt[0:2]) {
	case 0x10b: // PE32
	case 0x20b: // PE32+
		pe32Plus = true
	default:
		return errors.New("unknown PE optional header magic")
	}

	// Subsystem field position:
	//   PE32  : offset 68 within the optional header (16-bit)
	//   PE32+ : offset 68 within the optional header (16-bit)
	// Both share this offset: for the 16-bit Subsystem field it is at opt
	// header offset 0x44 (68). The optional header has been read 4 bytes
	// already; we need to skip to offset 68 - we read bytes 4..67.
	skip := 68 - 4
	buf := make([]byte, skip+2)
	if _, err := io.ReadFull(f, buf); err != nil {
		return fmt.Errorf("optional header read failed: %w", err)
	}
	subsystem := binary.LittleEndian.Uint16(buf[len(buf)-2:])
	_ = numSections
	if subsystem != imageSubsystemWindowsGUI && subsystem != imageSubsystemWindowsCUI {
		return fmt.Errorf("unsupported PE subsystem: 0x%04x", subsystem)
	}
	if subsystem != imageSubsystemWindowsGUI {
		return errors.New("PE is not a GUI subsystem (console window would show)")
	}
	_ = pe32Plus
	return nil
}

func validMachine(m uint16) bool {
	// IMAGE_FILE_MACHINE_AMD64 / I386 / ARM64 / ARM
	switch m {
	case 0x8664, 0x014c, 0xAA64, 0x01c0:
		return true
	}
	return false
}
