//go:build unit

package driver

import (
	"testing"
)

func TestIoctlQueryDevicePropertiesCode(t *testing.T) {
	// Calculate expected IOCTL code
	// Direction: Write (1), Type: 'g' (0x67), Nr: 1, Size: 32
	cmd := ioctlQueryDeviceProperties

	// Verify direction bits
	dir := (cmd >> IocDirShift) & 0x3
	if dir != IocWrite {
		t.Errorf("direction = %d, expected %d (write)", dir, IocWrite)
	}

	// Verify type
	typ := (cmd >> IocTypeShift) & 0xff
	if typ != uint32(HailoGeneralIoctlMagic) {
		t.Errorf("type = 0x%02x, expected 0x%02x", typ, HailoGeneralIoctlMagic)
	}

	// Verify nr
	nr := (cmd >> IocNrShift) & 0xff
	if nr != IoctlQueryDeviceProperties {
		t.Errorf("nr = %d, expected %d", nr, IoctlQueryDeviceProperties)
	}

	// Verify size
	size := (cmd >> IocSizeShift) & 0x3fff
	if size != uint32(SizeOfDeviceProperties) {
		t.Errorf("size = %d, expected %d", size, SizeOfDeviceProperties)
	}
}

func TestIoctlQueryDriverInfoCode(t *testing.T) {
	cmd := ioctlQueryDriverInfo

	dir := (cmd >> IocDirShift) & 0x3
	if dir != IocWrite {
		t.Errorf("direction = %d, expected %d (write)", dir, IocWrite)
	}

	typ := (cmd >> IocTypeShift) & 0xff
	if typ != uint32(HailoGeneralIoctlMagic) {
		t.Errorf("type = 0x%02x, expected 0x%02x", typ, HailoGeneralIoctlMagic)
	}

	nr := (cmd >> IocNrShift) & 0xff
	if nr != IoctlQueryDriverInfo {
		t.Errorf("nr = %d, expected %d", nr, IoctlQueryDriverInfo)
	}
}

func TestIoctlVdmaBufferMapCode(t *testing.T) {
	cmd := ioctlVdmaBufferMap

	// Should be read-write
	dir := (cmd >> IocDirShift) & 0x3
	if dir != (IocRead | IocWrite) {
		t.Errorf("direction = %d, expected %d (read|write)", dir, IocRead|IocWrite)
	}

	typ := (cmd >> IocTypeShift) & 0xff
	if typ != uint32(HailoVdmaIoctlMagic) {
		t.Errorf("type = 0x%02x, expected 0x%02x", typ, HailoVdmaIoctlMagic)
	}

	nr := (cmd >> IocNrShift) & 0xff
	if nr != IoctlVdmaBufferMap {
		t.Errorf("nr = %d, expected %d", nr, IoctlVdmaBufferMap)
	}
}

func TestIoctlVdmaEnableChannelsCode(t *testing.T) {
	cmd := ioctlVdmaEnableChannels

	// Should be read (data from user to kernel)
	dir := (cmd >> IocDirShift) & 0x3
	if dir != IocRead {
		t.Errorf("direction = %d, expected %d (read)", dir, IocRead)
	}

	typ := (cmd >> IocTypeShift) & 0xff
	if typ != uint32(HailoVdmaIoctlMagic) {
		t.Errorf("type = 0x%02x, expected 0x%02x", typ, HailoVdmaIoctlMagic)
	}
}

func TestIoctlVdmaInterruptsWaitCode(t *testing.T) {
	cmd := ioctlVdmaInterruptsWait

	// Should be read-write
	dir := (cmd >> IocDirShift) & 0x3
	if dir != (IocRead | IocWrite) {
		t.Errorf("direction = %d, expected %d (read|write)", dir, IocRead|IocWrite)
	}
}

func TestIoctlFwControlCode(t *testing.T) {
	cmd := ioctlFwControl

	// Should be read-write
	dir := (cmd >> IocDirShift) & 0x3
	if dir != (IocRead | IocWrite) {
		t.Errorf("direction = %d, expected %d (read|write)", dir, IocRead|IocWrite)
	}

	typ := (cmd >> IocTypeShift) & 0xff
	if typ != uint32(HailoNncIoctlMagic) {
		t.Errorf("type = 0x%02x, expected 0x%02x", typ, HailoNncIoctlMagic)
	}
}

func TestIoctlResetNnCoreCode(t *testing.T) {
	cmd := ioctlResetNnCore

	// Should be no direction (no data)
	dir := (cmd >> IocDirShift) & 0x3
	if dir != IocNone {
		t.Errorf("direction = %d, expected %d (none)", dir, IocNone)
	}

	// Size should be 0
	size := (cmd >> IocSizeShift) & 0x3fff
	if size != 0 {
		t.Errorf("size = %d, expected 0", size)
	}
}

func TestAllVdmaIoctlCodesUseVdmaMagic(t *testing.T) {
	vdmaCommands := []struct {
		name string
		cmd  uint32
	}{
		{"EnableChannels", ioctlVdmaEnableChannels},
		{"DisableChannels", ioctlVdmaDisableChannels},
		{"InterruptsWait", ioctlVdmaInterruptsWait},
		{"BufferMap", ioctlVdmaBufferMap},
		{"BufferUnmap", ioctlVdmaBufferUnmap},
		{"BufferSync", ioctlVdmaBufferSync},
		{"DescListCreate", ioctlDescListCreate},
		{"DescListRelease", ioctlDescListRelease},
		{"DescListProgram", ioctlDescListProgram},
		{"LaunchTransfer", ioctlVdmaLaunchTransfer},
	}

	for _, tt := range vdmaCommands {
		t.Run(tt.name, func(t *testing.T) {
			typ := (tt.cmd >> IocTypeShift) & 0xff
			if typ != uint32(HailoVdmaIoctlMagic) {
				t.Errorf("type = 0x%02x, expected 0x%02x", typ, HailoVdmaIoctlMagic)
			}
		})
	}
}

func TestAllNncIoctlCodesUseNncMagic(t *testing.T) {
	nncCommands := []struct {
		name string
		cmd  uint32
	}{
		{"FwControl", ioctlFwControl},
		{"ReadNotification", ioctlReadNotification},
		{"ResetNnCore", ioctlResetNnCore},
	}

	for _, tt := range nncCommands {
		t.Run(tt.name, func(t *testing.T) {
			typ := (tt.cmd >> IocTypeShift) & 0xff
			if typ != uint32(HailoNncIoctlMagic) {
				t.Errorf("type = 0x%02x, expected 0x%02x", typ, HailoNncIoctlMagic)
			}
		})
	}
}

func TestIoctlCodesAreUnique(t *testing.T) {
	// Collect all IOCTL codes
	codes := map[uint32]string{
		ioctlQueryDeviceProperties: "QueryDeviceProperties",
		ioctlQueryDriverInfo:       "QueryDriverInfo",
		ioctlVdmaEnableChannels:    "VdmaEnableChannels",
		ioctlVdmaDisableChannels:   "VdmaDisableChannels",
		ioctlVdmaInterruptsWait:    "VdmaInterruptsWait",
		ioctlVdmaBufferMap:         "VdmaBufferMap",
		ioctlVdmaBufferUnmap:       "VdmaBufferUnmap",
		ioctlVdmaBufferSync:        "VdmaBufferSync",
		ioctlDescListCreate:        "DescListCreate",
		ioctlDescListRelease:       "DescListRelease",
		ioctlDescListProgram:       "DescListProgram",
		ioctlVdmaLaunchTransfer:    "VdmaLaunchTransfer",
		ioctlFwControl:             "FwControl",
		ioctlReadNotification:      "ReadNotification",
		ioctlResetNnCore:           "ResetNnCore",
	}

	// The map will have fewer entries if there are duplicates
	expectedCount := 15
	if len(codes) != expectedCount {
		t.Errorf("expected %d unique IOCTL codes, got %d (some codes are duplicated)", expectedCount, len(codes))
	}
}

func TestIoctlSizesAreReasonable(t *testing.T) {
	// All IOCTL parameter sizes should be reasonable (< 64KB)
	codes := []struct {
		name string
		cmd  uint32
	}{
		{"QueryDeviceProperties", ioctlQueryDeviceProperties},
		{"QueryDriverInfo", ioctlQueryDriverInfo},
		{"VdmaEnableChannels", ioctlVdmaEnableChannels},
		{"VdmaBufferMap", ioctlVdmaBufferMap},
		{"FwControl", ioctlFwControl},
		{"VdmaInterruptsWait", ioctlVdmaInterruptsWait},
	}

	for _, tt := range codes {
		t.Run(tt.name, func(t *testing.T) {
			size := (tt.cmd >> IocSizeShift) & 0x3fff
			if size > 16384 { // 16KB is a reasonable maximum
				t.Errorf("size = %d bytes, seems too large", size)
			}
		})
	}
}
