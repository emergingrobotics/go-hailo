//go:build unit

package driver

import (
	"reflect"
	"testing"
	"unsafe"
)

// These expected sizes are based on the C struct definitions on arm64/amd64
// They need to match exactly for IOCTL to work correctly

func TestDevicePropertiesSize(t *testing.T) {
	// struct hailo_device_properties on 64-bit:
	// uint16_t desc_max_page_size;    // 2 bytes + 2 padding
	// enum hailo_board_type;          // 4 bytes (uint32)
	// enum hailo_allocation_mode;     // 4 bytes (uint32)
	// enum hailo_dma_type;            // 4 bytes (uint32)
	// size_t dma_engines_count;       // 8 bytes (uint64)
	// bool is_fw_loaded;              // 1 byte + 7 padding
	// Total: 32 bytes
	expected := 32
	got := SizeOfDeviceProperties
	if got != expected {
		t.Errorf("DeviceProperties size = %d, expected %d", got, expected)
	}
}

func TestDevicePropertiesFieldOffsets(t *testing.T) {
	var p DeviceProperties
	base := uintptr(unsafe.Pointer(&p))

	tests := []struct {
		name     string
		field    uintptr
		expected uintptr
	}{
		{"DescMaxPageSize", uintptr(unsafe.Pointer(&p.DescMaxPageSize)) - base, 0},
		{"BoardType", uintptr(unsafe.Pointer(&p.BoardType)) - base, 4},
		{"AllocationMode", uintptr(unsafe.Pointer(&p.AllocationMode)) - base, 8},
		{"DmaType", uintptr(unsafe.Pointer(&p.DmaType)) - base, 12},
		{"DmaEnginesCount", uintptr(unsafe.Pointer(&p.DmaEnginesCount)) - base, 16},
		{"IsFwLoaded", uintptr(unsafe.Pointer(&p.IsFwLoaded)) - base, 24},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field != tt.expected {
				t.Errorf("offset of %s = %d, expected %d", tt.name, tt.field, tt.expected)
			}
		})
	}
}

func TestDriverInfoSize(t *testing.T) {
	// 3 x uint32 = 12 bytes
	expected := 12
	got := SizeOfDriverInfo
	if got != expected {
		t.Errorf("DriverInfo size = %d, expected %d", got, expected)
	}
}

func TestVdmaBufferMapParamsSize(t *testing.T) {
	// uintptr user_address;           // 8 bytes
	// uint64 size;                    // 8 bytes
	// uint32 data_direction;          // 4 bytes
	// uint32 buffer_type;             // 4 bytes
	// uintptr allocated_buffer_handle;// 8 bytes
	// uint64 mapped_handle;           // 8 bytes
	// Total: 40 bytes
	expected := 40
	got := SizeOfVdmaBufferMapParams
	if got != expected {
		t.Errorf("VdmaBufferMapParams size = %d, expected %d", got, expected)
	}
}

func TestVdmaBufferMapParamsFieldOffsets(t *testing.T) {
	var p VdmaBufferMapParams
	base := uintptr(unsafe.Pointer(&p))

	tests := []struct {
		name     string
		field    uintptr
		expected uintptr
	}{
		{"UserAddress", uintptr(unsafe.Pointer(&p.UserAddress)) - base, 0},
		{"Size", uintptr(unsafe.Pointer(&p.Size)) - base, 8},
		{"DataDirection", uintptr(unsafe.Pointer(&p.DataDirection)) - base, 16},
		{"BufferType", uintptr(unsafe.Pointer(&p.BufferType)) - base, 20},
		{"AllocatedBufferHandle", uintptr(unsafe.Pointer(&p.AllocatedBufferHandle)) - base, 24},
		{"MappedHandle", uintptr(unsafe.Pointer(&p.MappedHandle)) - base, 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field != tt.expected {
				t.Errorf("offset of %s = %d, expected %d", tt.name, tt.field, tt.expected)
			}
		})
	}
}

func TestVdmaBufferSyncParamsSize(t *testing.T) {
	// uint64 handle + uint32 sync_type + padding + uint64 offset + uint64 count
	expected := 32
	got := SizeOfVdmaBufferSyncParams
	if got != expected {
		t.Errorf("VdmaBufferSyncParams size = %d, expected %d", got, expected)
	}
}

func TestDescListCreateParamsSize(t *testing.T) {
	// uint64 desc_count;      // 8 bytes
	// uint16 desc_page_size;  // 2 bytes
	// bool is_circular;       // 1 byte + 5 padding
	// uintptr desc_handle;    // 8 bytes
	// uint64 dma_address;     // 8 bytes
	// Total: 32 bytes
	expected := 32
	got := SizeOfDescListCreateParams
	if got != expected {
		t.Errorf("DescListCreateParams size = %d, expected %d", got, expected)
	}
}

func TestVdmaEnableChannelsParamsSize(t *testing.T) {
	// [3]uint32 + bool + padding = 16 bytes
	expected := 16
	got := SizeOfVdmaEnableChannelsParams
	if got != expected {
		t.Errorf("VdmaEnableChannelsParams size = %d, expected %d", got, expected)
	}
}

func TestVdmaDisableChannelsParamsSize(t *testing.T) {
	// [3]uint32 = 12 bytes
	expected := 12
	got := SizeOfVdmaDisableChannelsParams
	if got != expected {
		t.Errorf("VdmaDisableChannelsParams size = %d, expected %d", got, expected)
	}
}

func TestChannelInterruptTimestampSize(t *testing.T) {
	// uint64 timestamp_ns + uint16 desc_num + padding = 16 bytes
	expected := 16
	got := SizeOfChannelInterruptTimestamp
	if got != expected {
		t.Errorf("ChannelInterruptTimestamp size = %d, expected %d", got, expected)
	}
}

func TestVdmaTransferBufferSize(t *testing.T) {
	// uint32 buffer_type + padding + uintptr addr + uint32 size + padding
	expected := 24
	got := SizeOfVdmaTransferBuffer
	if got != expected {
		t.Errorf("VdmaTransferBuffer size = %d, expected %d", got, expected)
	}
}

func TestFwControlSize(t *testing.T) {
	// [16]byte md5 + uint32 buffer_len + [1500]byte buffer + uint32 timeout + uint32 cpu_id
	// = 16 + 4 + 1500 + 4 + 4 = 1528 bytes
	expected := 1528
	got := SizeOfFwControl
	if got != expected {
		t.Errorf("FwControl size = %d, expected %d", got, expected)
	}
}

func TestD2hNotificationSize(t *testing.T) {
	// uint64 buffer_len + [1500]byte buffer = 1508 bytes (with possible padding to 1512)
	got := SizeOfD2hNotification
	if got < 1508 {
		t.Errorf("D2hNotification size = %d, expected at least 1508", got)
	}
}

func TestAllEnumsAreUint32(t *testing.T) {
	// In C, enums are typically int/uint32, verify our Go types match
	tests := []struct {
		name string
		size int
	}{
		{"BoardType", int(unsafe.Sizeof(BoardType(0)))},
		{"DmaType", int(unsafe.Sizeof(DmaType(0)))},
		{"DmaDataDirection", int(unsafe.Sizeof(DmaDataDirection(0)))},
		{"DmaBufferType", int(unsafe.Sizeof(DmaBufferType(0)))},
		{"AllocationMode", int(unsafe.Sizeof(AllocationMode(0)))},
		{"BufferSyncType", int(unsafe.Sizeof(BufferSyncType(0)))},
		{"InterruptsDomain", int(unsafe.Sizeof(InterruptsDomain(0)))},
		{"CpuId", int(unsafe.Sizeof(CpuId(0)))},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.size != 4 {
				t.Errorf("%s size = %d, expected 4 (uint32)", tt.name, tt.size)
			}
		})
	}
}

func TestVdmaInterruptsWaitParamsArraySizes(t *testing.T) {
	var p VdmaInterruptsWaitParams

	// Check ChannelsBitmapPerEngine array size
	if len(p.ChannelsBitmapPerEngine) != MaxVdmaEngines {
		t.Errorf("ChannelsBitmapPerEngine length = %d, expected %d",
			len(p.ChannelsBitmapPerEngine), MaxVdmaEngines)
	}

	// Check IrqData array size
	expectedIrqDataLen := MaxVdmaChannelsPerEngine * MaxVdmaEngines
	if len(p.IrqData) != expectedIrqDataLen {
		t.Errorf("IrqData length = %d, expected %d",
			len(p.IrqData), expectedIrqDataLen)
	}
}

func TestVdmaInterruptsReadTimestampParamsArraySize(t *testing.T) {
	var p VdmaInterruptsReadTimestampParams

	if len(p.Timestamps) != ChannelIrqTimestampsSize {
		t.Errorf("Timestamps length = %d, expected %d",
			len(p.Timestamps), ChannelIrqTimestampsSize)
	}
}

func TestVdmaLaunchTransferParamsArraySize(t *testing.T) {
	var p VdmaLaunchTransferParams

	if len(p.Buffers) != HailoMaxBuffersPerSingleTransfer {
		t.Errorf("Buffers length = %d, expected %d",
			len(p.Buffers), HailoMaxBuffersPerSingleTransfer)
	}
}

func TestFwControlArraySizes(t *testing.T) {
	var p FwControl

	if len(p.ExpectedMd5) != PcieExpectedMd5Length {
		t.Errorf("ExpectedMd5 length = %d, expected %d",
			len(p.ExpectedMd5), PcieExpectedMd5Length)
	}

	if len(p.Buffer) != MaxControlLength {
		t.Errorf("Buffer length = %d, expected %d",
			len(p.Buffer), MaxControlLength)
	}
}

func TestD2hNotificationArraySize(t *testing.T) {
	var p D2hNotification

	if len(p.Buffer) != MaxNotificationLength {
		t.Errorf("Buffer length = %d, expected %d",
			len(p.Buffer), MaxNotificationLength)
	}
}

func TestReadLogParamsArraySize(t *testing.T) {
	var p ReadLogParams

	if len(p.Buffer) != MaxFwLogBufferLength {
		t.Errorf("Buffer length = %d, expected %d",
			len(p.Buffer), MaxFwLogBufferLength)
	}
}

func TestStructsHaveNoUnexportedPublicFields(t *testing.T) {
	// Verify that exported structs only have expected fields
	// (no accidental unexported fields that would cause issues)

	types := []reflect.Type{
		reflect.TypeOf(DeviceProperties{}),
		reflect.TypeOf(DriverInfo{}),
		reflect.TypeOf(VdmaBufferMapParams{}),
		reflect.TypeOf(FwControl{}),
	}

	for _, typ := range types {
		t.Run(typ.Name(), func(t *testing.T) {
			for i := 0; i < typ.NumField(); i++ {
				field := typ.Field(i)
				// Padding fields start with _ and are unexported, which is fine
				if field.Name == "_" || field.Name[0] == '_' {
					continue
				}
				// All other fields should be exported (start with uppercase)
				if field.Name[0] >= 'a' && field.Name[0] <= 'z' {
					t.Errorf("unexpected unexported field %s in %s", field.Name, typ.Name())
				}
			}
		})
	}
}

func TestUintptrSizeIs8OnAmd64Arm64(t *testing.T) {
	// This test ensures we're testing on a 64-bit platform
	// where uintptr is 8 bytes (required for correct struct layout)
	size := unsafe.Sizeof(uintptr(0))
	if size != 8 {
		t.Skipf("Skipping 64-bit specific tests, uintptr size = %d", size)
	}
}

func TestDescListProgramParamsSize(t *testing.T) {
	// Complex struct with many fields, verify total size
	got := SizeOfDescListProgramParams
	// Needs to be verified against C struct
	if got == 0 {
		t.Error("DescListProgramParams size should not be 0")
	}
	t.Logf("DescListProgramParams size = %d", got)
}

func TestVdmaInterruptsWaitParamsSize(t *testing.T) {
	got := SizeOfVdmaInterruptsWaitParams
	// [3]uint32 + uint8 + padding + [96]VdmaInterruptsChannelData
	// = 12 + 4 + (96 * 4) = 400 bytes approximately
	if got < 400 {
		t.Errorf("VdmaInterruptsWaitParams size = %d, expected >= 400", got)
	}
	t.Logf("VdmaInterruptsWaitParams size = %d", got)
}

func TestVdmaLaunchTransferParamsSize(t *testing.T) {
	got := SizeOfVdmaLaunchTransferParams
	// Complex struct, verify it's a reasonable size
	// Driver 4.20.0: 2 Buffers * 24 bytes each = 48 + other fields ~40 = ~88
	// Driver 4.23.0: 8 Buffers * 24 bytes each = 192 + other fields ~40 = ~232
	if got < 60 {
		t.Errorf("VdmaLaunchTransferParams size = %d, expected >= 60", got)
	}
	t.Logf("VdmaLaunchTransferParams size = %d", got)
}
