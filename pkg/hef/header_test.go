//go:build unit

package hef

import (
	"encoding/binary"
	"errors"
	"testing"
)

func makeValidHeader(version uint32) []byte {
	size := 56 // Maximum header size
	data := make([]byte, size)

	binary.LittleEndian.PutUint32(data[0:4], HefMagic)
	binary.LittleEndian.PutUint32(data[4:8], version)
	binary.LittleEndian.PutUint32(data[8:12], 1000) // HefProtoSize

	return data
}

func TestValidMagicNumber(t *testing.T) {
	data := makeValidHeader(HefVersionV2)

	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if header.Magic != HefMagic {
		t.Errorf("magic = 0x%08X, expected 0x%08X", header.Magic, HefMagic)
	}
}

func TestInvalidMagicNumber(t *testing.T) {
	data := make([]byte, 56)
	binary.LittleEndian.PutUint32(data[0:4], 0xDEADBEEF)
	binary.LittleEndian.PutUint32(data[4:8], HefVersionV2)

	_, err := ParseHeader(data)
	if err == nil {
		t.Error("expected error for invalid magic number")
	}

	if !errors.Is(err, ErrInvalidMagic) {
		t.Errorf("expected ErrInvalidMagic, got %v", err)
	}
}

func TestVersionZeroHeader(t *testing.T) {
	data := makeValidHeader(HefVersionV0)

	// Add MD5 for V0
	for i := 0; i < 16; i++ {
		data[16+i] = byte(i)
	}

	header, err := ParseHeaderV0(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if header.Version != HefVersionV0 {
		t.Errorf("version = %d, expected %d", header.Version, HefVersionV0)
	}

	// Verify MD5 was read
	for i := 0; i < 16; i++ {
		if header.ExpectedMD5[i] != byte(i) {
			t.Errorf("ExpectedMD5[%d] = %d, expected %d", i, header.ExpectedMD5[i], i)
		}
	}
}

func TestVersionTwoHeader(t *testing.T) {
	data := makeValidHeader(HefVersionV2)

	// Set XXH3 hash
	binary.LittleEndian.PutUint64(data[12:20], 0x123456789ABCDEF0)
	// Set CCWS size
	binary.LittleEndian.PutUint64(data[20:28], 50000)

	header, err := ParseHeaderV2(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if header.Version != HefVersionV2 {
		t.Errorf("version = %d, expected %d", header.Version, HefVersionV2)
	}

	if header.Xxh3Hash != 0x123456789ABCDEF0 {
		t.Errorf("Xxh3Hash = 0x%016X, expected 0x123456789ABCDEF0", header.Xxh3Hash)
	}

	if header.CcwsSize != 50000 {
		t.Errorf("CcwsSize = %d, expected 50000", header.CcwsSize)
	}
}

func TestVersionThreeHeader(t *testing.T) {
	data := makeValidHeader(HefVersionV3)

	// Set XXH3 hash
	binary.LittleEndian.PutUint64(data[12:20], 0xFEDCBA9876543210)
	// Set CCWS size with padding
	binary.LittleEndian.PutUint64(data[20:28], 100000)
	// Set padding size
	binary.LittleEndian.PutUint32(data[28:32], 256)
	// Set additional info size
	binary.LittleEndian.PutUint64(data[36:44], 1024)

	header, err := ParseHeaderV3(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if header.Version != HefVersionV3 {
		t.Errorf("version = %d, expected %d", header.Version, HefVersionV3)
	}

	if header.Xxh3Hash != 0xFEDCBA9876543210 {
		t.Errorf("Xxh3Hash = 0x%016X, expected 0xFEDCBA9876543210", header.Xxh3Hash)
	}

	if header.CcwsSizeWithPadding != 100000 {
		t.Errorf("CcwsSizeWithPadding = %d, expected 100000", header.CcwsSizeWithPadding)
	}

	if header.HefPaddingSize != 256 {
		t.Errorf("HefPaddingSize = %d, expected 256", header.HefPaddingSize)
	}

	if header.AdditionalInfoSize != 1024 {
		t.Errorf("AdditionalInfoSize = %d, expected 1024", header.AdditionalInfoSize)
	}
}

func TestUnknownVersionRejected(t *testing.T) {
	data := makeValidHeader(99)

	_, err := HeaderSize(99)
	if err == nil {
		t.Error("expected error for unknown version")
	}

	if !errors.Is(err, ErrUnsupportedVersion) {
		t.Errorf("expected ErrUnsupportedVersion, got %v", err)
	}

	// Also test that version-specific parsers reject wrong versions
	_, err = ParseHeaderV0(data)
	if err == nil {
		t.Error("expected error when parsing V99 as V0")
	}
}

func TestTruncatedHeaderRejected(t *testing.T) {
	testCases := []struct {
		name     string
		dataSize int
	}{
		{"empty", 0},
		{"too short", 8},
		{"just under minimum", 11},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			data := make([]byte, tc.dataSize)

			_, err := ParseHeader(data)
			if err == nil {
				t.Error("expected error for truncated header")
			}

			if !errors.Is(err, ErrTruncatedHeader) {
				t.Errorf("expected ErrTruncatedHeader, got %v", err)
			}
		})
	}
}

func TestHeaderSizeByVersion(t *testing.T) {
	testCases := []struct {
		version  uint32
		expected int
	}{
		{HefVersionV0, HefHeaderSizeV0},
		{HefVersionV1, HefHeaderSizeV1},
		{HefVersionV2, HefHeaderSizeV2},
		{HefVersionV3, HefHeaderSizeV3},
	}

	for _, tc := range testCases {
		t.Run("V"+string(rune('0'+tc.version)), func(t *testing.T) {
			size, err := HeaderSize(tc.version)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if size != tc.expected {
				t.Errorf("HeaderSize(V%d) = %d, expected %d", tc.version, size, tc.expected)
			}
		})
	}
}

func TestHefMagicValue(t *testing.T) {
	// HEF magic should be "FEH" followed by 0x01 in little-endian
	// That's 0x46 ('F'), 0x45 ('E'), 0x48 ('H'), 0x01
	// In little-endian uint32: 0x01484546

	if HefMagic != 0x01484546 {
		t.Errorf("HefMagic = 0x%08X, expected 0x01484546", HefMagic)
	}

	// Verify bytes
	var buf [4]byte
	binary.LittleEndian.PutUint32(buf[:], HefMagic)
	expected := []byte{'F', 'E', 'H', 0x01}
	for i, b := range expected {
		if buf[i] != b {
			t.Errorf("magic byte[%d] = 0x%02X, expected 0x%02X", i, buf[i], b)
		}
	}
}

func TestHeaderConstants(t *testing.T) {
	// Verify header size constants are consistent
	if HefHeaderSizeV0 != 32 {
		t.Errorf("HefHeaderSizeV0 = %d, expected 32", HefHeaderSizeV0)
	}
	if HefHeaderSizeV1 != 32 {
		t.Errorf("HefHeaderSizeV1 = %d, expected 32", HefHeaderSizeV1)
	}
	if HefHeaderSizeV2 != 40 {
		t.Errorf("HefHeaderSizeV2 = %d, expected 40", HefHeaderSizeV2)
	}
	if HefHeaderSizeV3 != 56 {
		t.Errorf("HefHeaderSizeV3 = %d, expected 56", HefHeaderSizeV3)
	}
}

func TestVersionV0ParserRejectsV2Data(t *testing.T) {
	data := makeValidHeader(HefVersionV2)

	_, err := ParseHeaderV0(data)
	if err == nil {
		t.Error("V0 parser should reject V2 data")
	}
}

func TestVersionV2ParserRejectsV0Data(t *testing.T) {
	data := makeValidHeader(HefVersionV0)

	_, err := ParseHeaderV2(data)
	if err == nil {
		t.Error("V2 parser should reject V0 data")
	}
}

func TestVersionV3ParserRejectsV2Data(t *testing.T) {
	data := makeValidHeader(HefVersionV2)

	_, err := ParseHeaderV3(data)
	if err == nil {
		t.Error("V3 parser should reject V2 data")
	}
}

func TestProtoSizeExtraction(t *testing.T) {
	data := makeValidHeader(HefVersionV2)
	binary.LittleEndian.PutUint32(data[8:12], 12345)

	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if header.HefProtoSize != 12345 {
		t.Errorf("HefProtoSize = %d, expected 12345", header.HefProtoSize)
	}
}

func TestMinimumValidHeader(t *testing.T) {
	// Create absolute minimum valid header (12 bytes for common part)
	data := make([]byte, 12)
	binary.LittleEndian.PutUint32(data[0:4], HefMagic)
	binary.LittleEndian.PutUint32(data[4:8], HefVersionV0)
	binary.LittleEndian.PutUint32(data[8:12], 100)

	header, err := ParseHeader(data)
	if err != nil {
		t.Fatalf("unexpected error for minimum valid header: %v", err)
	}

	if header.Magic != HefMagic {
		t.Error("magic mismatch")
	}
	if header.Version != HefVersionV0 {
		t.Error("version mismatch")
	}
	if header.HefProtoSize != 100 {
		t.Error("proto size mismatch")
	}
}
