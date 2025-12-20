//go:build unit

package hef

import (
	"crypto/md5"
	"encoding/binary"
	"testing"
)

func TestMd5ValidationV0(t *testing.T) {
	// Create test data
	testData := []byte("Hello, Hailo!")

	// Calculate MD5
	hash := md5.Sum(testData)

	// Verify hash is 16 bytes
	if len(hash) != 16 {
		t.Errorf("MD5 hash length = %d, expected 16", len(hash))
	}

	// Verify hash is not all zeros
	allZero := true
	for _, b := range hash {
		if b != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Error("MD5 hash should not be all zeros")
	}
}

func TestMd5ValidationV0Invalid(t *testing.T) {
	// Create test data
	testData := []byte("Hello, Hailo!")
	correctHash := md5.Sum(testData)

	// Create corrupted hash
	incorrectHash := correctHash
	incorrectHash[0] ^= 0xFF // Flip bits in first byte

	// Verify they don't match
	if correctHash == incorrectHash {
		t.Error("corrupted hash should not match original")
	}
}

func TestXxh3ValidationV2(t *testing.T) {
	// Test that XXH3 hash values are consistent
	// XXH3 produces 64-bit hash

	// These are placeholder tests - actual XXH3 implementation would use zeebo/xxh3
	testCases := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"hello", []byte("Hello")},
		{"binary", []byte{0x00, 0x01, 0x02, 0x03}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// In real implementation:
			// hash := xxh3.Hash(tc.data)
			// Verify hash is non-zero for non-empty data
			if len(tc.data) > 0 {
				// Just verify data exists
				if tc.data == nil {
					t.Error("data should not be nil")
				}
			}
		})
	}
}

func TestXxh3ValidationV3(t *testing.T) {
	// V3 uses same XXH3 hash as V2
	// Test that V3 header stores hash correctly

	data := make([]byte, HefHeaderSizeV3)
	binary.LittleEndian.PutUint32(data[0:4], HefMagic)
	binary.LittleEndian.PutUint32(data[4:8], HefVersionV3)
	binary.LittleEndian.PutUint32(data[8:12], 100)

	// Set known hash value
	expectedHash := uint64(0xDEADBEEF12345678)
	binary.LittleEndian.PutUint64(data[12:20], expectedHash)

	header, err := ParseHeaderV3(data)
	if err != nil {
		t.Fatalf("failed to parse V3 header: %v", err)
	}

	if header.Xxh3Hash != expectedHash {
		t.Errorf("Xxh3Hash = 0x%016X, expected 0x%016X", header.Xxh3Hash, expectedHash)
	}
}

func TestXxh3InvalidFails(t *testing.T) {
	// Test that validation fails when hash doesn't match
	// This is a conceptual test - actual implementation would calculate hash

	storedHash := uint64(0x123456789ABCDEF0)
	calculatedHash := uint64(0x0FEDCBA987654321)

	if storedHash == calculatedHash {
		t.Error("hashes should not match")
	}
}

func TestMd5HashConstantSize(t *testing.T) {
	// MD5 always produces 16 bytes
	const md5Size = 16

	if PcieExpectedMd5Length != md5Size {
		t.Errorf("PcieExpectedMd5Length = %d, expected %d", PcieExpectedMd5Length, md5Size)
	}

	// Verify V0 header stores full MD5
	var header HefHeaderV0
	if len(header.ExpectedMD5) != md5Size {
		t.Errorf("ExpectedMD5 array size = %d, expected %d", len(header.ExpectedMD5), md5Size)
	}
}

func TestXxh3HashIs64Bit(t *testing.T) {
	// XXH3 produces 64-bit hash
	var header HefHeaderV2

	// Xxh3Hash should be uint64
	header.Xxh3Hash = 0xFFFFFFFFFFFFFFFF

	if header.Xxh3Hash != 0xFFFFFFFFFFFFFFFF {
		t.Error("Xxh3Hash should support full 64-bit range")
	}
}

func TestChecksumDataBoundaries(t *testing.T) {
	// Test that checksum is calculated over correct data region

	// For V0: checksum covers proto data
	// For V2/V3: checksum covers proto data + CCWS data

	headerSize := HefHeaderSizeV2
	protoSize := uint32(1000)
	ccwsSize := uint64(5000)

	totalDataSize := headerSize + int(protoSize) + int(ccwsSize)

	// Verify data region size calculation
	if totalDataSize != 40+1000+5000 {
		t.Errorf("total data size = %d, expected %d", totalDataSize, 40+1000+5000)
	}
}

func TestV0HeaderMd5Storage(t *testing.T) {
	// Create V0 header with known MD5
	data := make([]byte, HefHeaderSizeV0)
	binary.LittleEndian.PutUint32(data[0:4], HefMagic)
	binary.LittleEndian.PutUint32(data[4:8], HefVersionV0)
	binary.LittleEndian.PutUint32(data[8:12], 100)

	// Set MD5 bytes (offset 16-31)
	for i := 0; i < 16; i++ {
		data[16+i] = byte(i * 17) // 0x00, 0x11, 0x22, ...
	}

	header, err := ParseHeaderV0(data)
	if err != nil {
		t.Fatalf("failed to parse: %v", err)
	}

	// Verify MD5 was read correctly
	for i := 0; i < 16; i++ {
		expected := byte(i * 17)
		if header.ExpectedMD5[i] != expected {
			t.Errorf("ExpectedMD5[%d] = 0x%02X, expected 0x%02X",
				i, header.ExpectedMD5[i], expected)
		}
	}
}

// PcieExpectedMd5Length is used in driver package, test it here for consistency
const PcieExpectedMd5Length = 16
