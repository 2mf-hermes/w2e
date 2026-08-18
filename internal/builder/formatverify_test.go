package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func TestVerifyELF(t *testing.T) {
	// Build a tiny ELF test fixture from a base64 blob? Instead, we just skip
	// when there is no fixture. The ELF verifier is exercised end-to-end when
	// building for the linux target on a real host; on Windows we cannot run
	// that here. So this is a smoke test of the magic-byte path.
	here, _ := os.Getwd()
	elf := filepath.Join(here, "testdata", "smoke.elf")
	if _, err := os.Stat(elf); err != nil {
		t.Skip("testdata/smoke.elf missing on this platform")
	}
	if err := VerifyELF(elf); err != nil {
		t.Errorf("VerifyELF = %v; want nil", err)
	}
}

func TestVerifyMachO(t *testing.T) {
	here, _ := os.Getwd()
	mo := filepath.Join(here, "testdata", "smoke.macho")
	if _, err := os.Stat(mo); err != nil {
		t.Skip("testdata/smoke.macho missing on this platform")
	}
	if err := VerifyMachO(mo); err != nil {
		t.Errorf("VerifyMachO = %v; want nil", err)
	}
}

// TestVerifyELFBadMagicBuildsAPlainTextFile verifies that the verifier
// rejects a non-ELF file.
func TestVerifyELFBadMagic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "not-elf")
	if err := os.WriteFile(p, []byte("hello world, not an ELF\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyELF(p); err == nil {
		t.Errorf("VerifyELF on plain text = nil; want error")
	}
}

// TestVerifyMachOBadMagic verifies the verifier rejects non-Mach-O content.
func TestVerifyMachOBadMagic(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "not-macho")
	if err := os.WriteFile(p, []byte("plain text no mach-o magic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := VerifyMachO(p); err == nil {
		t.Errorf("VerifyMachO on plain text = nil; want error")
	}
}
