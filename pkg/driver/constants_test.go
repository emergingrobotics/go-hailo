//go:build unit

package driver

import (
	"testing"
)

func TestIoctlMagicValues(t *testing.T) {
	tests := []struct {
		name     string
		got      byte
		expected byte
	}{
		{"GeneralMagic", HailoGeneralIoctlMagic, 0x67},
		{"VdmaMagic", HailoVdmaIoctlMagic, 0x76},
		{"NncMagic", HailoNncIoctlMagic, 0x6e},
		{"SocMagic", HailoSocIoctlMagic, 0x73},
		{"PciEpMagic", HailoPciEpIoctlMagic, 0x70},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected 0x%02x, got 0x%02x", tt.expected, tt.got)
			}
		})
	}
}

func TestDriverVersionConstants(t *testing.T) {
	if HailoDrvVerMajor != 4 {
		t.Errorf("expected major version 4, got %d", HailoDrvVerMajor)
	}
	if HailoDrvVerMinor != 20 {
		t.Errorf("expected minor version 20, got %d", HailoDrvVerMinor)
	}
	if HailoDrvVerRevision != 0 {
		t.Errorf("expected revision 0, got %d", HailoDrvVerRevision)
	}
}

func TestVdmaConstants(t *testing.T) {
	tests := []struct {
		name     string
		got      int
		expected int
	}{
		{"MaxVdmaChannelsPerEngine", MaxVdmaChannelsPerEngine, 32},
		{"VdmaChannelsPerEnginePerDirection", VdmaChannelsPerEnginePerDirection, 16},
		{"MaxVdmaEngines", MaxVdmaEngines, 3},
		{"SizeOfVdmaDescriptor", SizeOfVdmaDescriptor, 16},
		{"VdmaDestChannelsStart", VdmaDestChannelsStart, 16},
		{"MaxSgDescsCount", MaxSgDescsCount, 64 * 1024},
		{"HailoVdmaMaxOngoingTransfers", HailoVdmaMaxOngoingTransfers, 128},
		{"ChannelIrqTimestampsSize", ChannelIrqTimestampsSize, 256},
		{"MaxControlLength", MaxControlLength, 1500},
		{"HailoMaxBuffersPerSingleTransfer", HailoMaxBuffersPerSingleTransfer, 2}, // 2 in 4.20.0
		{"PcieExpectedMd5Length", PcieExpectedMd5Length, 16},
		{"MaxFwLogBufferLength", MaxFwLogBufferLength, 512},
		{"MaxNotificationLength", MaxNotificationLength, 1500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, tt.got)
			}
		})
	}
}

func TestTransferDataSpecialValues(t *testing.T) {
	if HailoVdmaTransferDataChannelNotActive != 0xff {
		t.Errorf("expected 0xff, got 0x%02x", HailoVdmaTransferDataChannelNotActive)
	}
	if HailoVdmaTransferDataChannelWithError != 0xfe {
		t.Errorf("expected 0xfe, got 0x%02x", HailoVdmaTransferDataChannelWithError)
	}
}

func TestBoardTypeValues(t *testing.T) {
	tests := []struct {
		name     string
		got      BoardType
		expected BoardType
	}{
		{"Hailo8", BoardTypeHailo8, 0},
		{"Hailo15", BoardTypeHailo15, 1},
		{"Hailo15L", BoardTypeHailo15L, 2},
		{"Hailo10H", BoardTypeHailo10H, 3},
		{"Hailo10Legacy", BoardTypeHailo10Legacy, 4},
		{"Mars", BoardTypeMars, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, tt.got)
			}
		})
	}
}

func TestDmaTypeValues(t *testing.T) {
	if DmaTypePcie != 0 {
		t.Errorf("expected DmaTypePcie=0, got %d", DmaTypePcie)
	}
	if DmaTypeDram != 1 {
		t.Errorf("expected DmaTypeDram=1, got %d", DmaTypeDram)
	}
	if DmaTypePciEp != 2 {
		t.Errorf("expected DmaTypePciEp=2, got %d", DmaTypePciEp)
	}
}

func TestDmaDataDirectionValues(t *testing.T) {
	if DmaBidirectional != 0 {
		t.Errorf("expected DmaBidirectional=0, got %d", DmaBidirectional)
	}
	if DmaToDevice != 1 {
		t.Errorf("expected DmaToDevice=1, got %d", DmaToDevice)
	}
	if DmaFromDevice != 2 {
		t.Errorf("expected DmaFromDevice=2, got %d", DmaFromDevice)
	}
	if DmaNone != 3 {
		t.Errorf("expected DmaNone=3, got %d", DmaNone)
	}
}

func TestInterruptsDomainValues(t *testing.T) {
	if InterruptsDomainNone != 0 {
		t.Errorf("expected InterruptsDomainNone=0, got %d", InterruptsDomainNone)
	}
	if InterruptsDomainDevice != 1 {
		t.Errorf("expected InterruptsDomainDevice=1, got %d", InterruptsDomainDevice)
	}
	if InterruptsDomainHost != 2 {
		t.Errorf("expected InterruptsDomainHost=2, got %d", InterruptsDomainHost)
	}
}

func TestCpuIdValues(t *testing.T) {
	if CpuIdCpu0 != 0 {
		t.Errorf("expected CpuIdCpu0=0, got %d", CpuIdCpu0)
	}
	if CpuIdCpu1 != 1 {
		t.Errorf("expected CpuIdCpu1=1, got %d", CpuIdCpu1)
	}
	if CpuIdNone != 2 {
		t.Errorf("expected CpuIdNone=2, got %d", CpuIdNone)
	}
}

func TestIocMacro(t *testing.T) {
	// Test the Ioc macro produces correct bit layout
	// Based on Linux IOCTL encoding: direction(2) | size(14) | type(8) | nr(8)

	// Simple case: no data, type='g' (0x67), nr=1
	cmd := Io(int(HailoGeneralIoctlMagic), 1)
	// Expected: dir=0, size=0, type=0x67, nr=1
	// = (0 << 30) | (0 << 16) | (0x67 << 8) | 1
	// = 0x00006701
	expected := uint32(0x00006701)
	if cmd != expected {
		t.Errorf("Io('g', 1) = 0x%08x, expected 0x%08x", cmd, expected)
	}
}

func TestIoWMacro(t *testing.T) {
	// Write IOCTL: direction = 1
	// Type='v' (0x76), nr=4 (buffer map), size=48 (example)
	cmd := IoW(int(HailoVdmaIoctlMagic), 4, 48)

	// Verify direction bits are set for write
	dirBits := (cmd >> IocDirShift) & 0x3
	if dirBits != IocWrite {
		t.Errorf("IoW direction bits = %d, expected %d", dirBits, IocWrite)
	}

	// Verify type
	typeBits := (cmd >> IocTypeShift) & 0xff
	if typeBits != uint32(HailoVdmaIoctlMagic) {
		t.Errorf("IoW type bits = 0x%02x, expected 0x%02x", typeBits, HailoVdmaIoctlMagic)
	}

	// Verify nr
	nrBits := (cmd >> IocNrShift) & 0xff
	if nrBits != 4 {
		t.Errorf("IoW nr bits = %d, expected 4", nrBits)
	}

	// Verify size
	sizeBits := (cmd >> IocSizeShift) & 0x3fff
	if sizeBits != 48 {
		t.Errorf("IoW size bits = %d, expected 48", sizeBits)
	}
}

func TestIoRMacro(t *testing.T) {
	// Read IOCTL: direction = 2
	cmd := IoR(int(HailoGeneralIoctlMagic), 1, 32)

	dirBits := (cmd >> IocDirShift) & 0x3
	if dirBits != IocRead {
		t.Errorf("IoR direction bits = %d, expected %d", dirBits, IocRead)
	}
}

func TestIoWRMacro(t *testing.T) {
	// Read-write IOCTL: direction = 3
	cmd := IoWR(int(HailoVdmaIoctlMagic), 2, 64)

	dirBits := (cmd >> IocDirShift) & 0x3
	if dirBits != (IocRead | IocWrite) {
		t.Errorf("IoWR direction bits = %d, expected %d", dirBits, IocRead|IocWrite)
	}
}

func TestIoctlCommandNumbers(t *testing.T) {
	// Verify IOCTL command numbers match expected values
	// These would be cross-checked against the C header

	// General IOCTLs
	if IoctlQueryDeviceProperties != 1 {
		t.Error("IoctlQueryDeviceProperties should be 1")
	}
	if IoctlQueryDriverInfo != 2 {
		t.Error("IoctlQueryDriverInfo should be 2")
	}

	// VDMA IOCTLs
	vdmaCommands := []struct {
		name     string
		got      int
		expected int
	}{
		{"EnableChannels", IoctlVdmaEnableChannels, 0},
		{"DisableChannels", IoctlVdmaDisableChannels, 1},
		{"InterruptsWait", IoctlVdmaInterruptsWait, 2},
		{"InterruptsReadTimestamp", IoctlVdmaInterruptsReadTimestamp, 3},
		{"BufferMap", IoctlVdmaBufferMap, 4},
		{"BufferUnmap", IoctlVdmaBufferUnmap, 5},
		{"BufferSync", IoctlVdmaBufferSync, 6},
		{"DescListCreate", IoctlDescListCreate, 7},
		{"DescListRelease", IoctlDescListRelease, 8},
		{"DescListProgram", IoctlDescListProgram, 9},
		{"LowMemoryBufferAlloc", IoctlVdmaLowMemoryBufferAlloc, 10},
		{"LowMemoryBufferFree", IoctlVdmaLowMemoryBufferFree, 11},
		{"MarkAsInUse", IoctlMarkAsInUse, 12},
		{"ContinuousBufferAlloc", IoctlVdmaContinuousBufferAlloc, 13},
		{"ContinuousBufferFree", IoctlVdmaContinuousBufferFree, 14},
		{"LaunchTransfer", IoctlVdmaLaunchTransfer, 15},
	}

	for _, tt := range vdmaCommands {
		t.Run("Vdma"+tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, tt.got)
			}
		})
	}

	// NNC IOCTLs
	nncCommands := []struct {
		name     string
		got      int
		expected int
	}{
		{"FwControl", IoctlFwControl, 0},
		{"ReadNotification", IoctlReadNotification, 1},
		{"DisableNotification", IoctlDisableNotification, 2},
		{"ReadLog", IoctlReadLog, 3},
		{"ResetNnCore", IoctlResetNnCore, 4},
		{"WriteActionList", IoctlWriteActionList, 5},
	}

	for _, tt := range nncCommands {
		t.Run("Nnc"+tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, tt.got)
			}
		})
	}
}

func TestIocShiftConstants(t *testing.T) {
	// Verify shift values match Linux standard IOCTL encoding
	if IocNrShift != 0 {
		t.Errorf("IocNrShift should be 0, got %d", IocNrShift)
	}
	if IocTypeShift != 8 {
		t.Errorf("IocTypeShift should be 8, got %d", IocTypeShift)
	}
	if IocSizeShift != 16 {
		t.Errorf("IocSizeShift should be 16, got %d", IocSizeShift)
	}
	if IocDirShift != 30 {
		t.Errorf("IocDirShift should be 30, got %d", IocDirShift)
	}
}

func TestAllIoctlCodesCalculate(t *testing.T) {
	// Verify no panic when calculating all IOCTL codes
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("panic during IOCTL code calculation: %v", r)
		}
	}()

	// Calculate some representative IOCTLs
	_ = IoW(int(HailoGeneralIoctlMagic), IoctlQueryDeviceProperties, 32)
	_ = IoWR(int(HailoVdmaIoctlMagic), IoctlVdmaBufferMap, 48)
	_ = Io(int(HailoNncIoctlMagic), IoctlResetNnCore)
	_ = IoR(int(HailoVdmaIoctlMagic), IoctlVdmaInterruptsWait, 128)
}
