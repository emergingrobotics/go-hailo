//go:build unit

package driver

import (
	"testing"
)

// TestPackedStructSizes verifies that packed struct sizes match the official
// Hailo driver hailo_ioctl_common.h definitions (with #pragma pack(1))
// NOTE: These sizes are for driver 4.20.0
func TestPackedStructSizes(t *testing.T) {
	tests := []struct {
		name     string
		got      int
		expected int
	}{
		// From hailo_ioctl_common.h with #pragma pack(1) - Driver 4.20.0
		{"PackedDescListProgramParams", SizeOfPackedDescListProgramParams, 43},        // No batch_size/stride in 4.20.0
		{"PackedVdmaLaunchTransferParams", SizeOfPackedVdmaLaunchTransferParams, 65},  // 2 buffers, has output fields
		{"PackedVdmaInterruptsWaitParams", SizeOfPackedVdmaInterruptsWaitParams, 685}, // 7 bytes per irq_data
		{"PackedVdmaTransferBuffer", 16, 16},                                          // mapped_handle(8) + offset(4) + size(4)
		{"PackedWriteActionListParams", SizeOfPackedWriteActionListParams, 24},
		{"PackedDescListCreateParams", SizeOfPackedDescListCreateParams, 27},
		{"PackedVdmaEnableChannelsParams", SizeOfPackedVdmaEnableChannelsParams, 13},
		{"PackedVdmaDisableChannelsParams", SizeOfPackedVdmaDisableChannelsParams, 12},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got != tc.expected {
				t.Errorf("%s size = %d, expected %d", tc.name, tc.got, tc.expected)
			}
		})
	}
}

// TestPackedVdmaTransferBufferLayout verifies the buffer layout matches the C struct
// Driver 4.20.0 layout: mapped_buffer_handle(8) + offset(4) + size(4)
func TestPackedVdmaTransferBufferLayout(t *testing.T) {
	buf := NewPackedVdmaTransferBuffer(0x123456789ABCDEF0, 0x11223344, 0xDEADBEEF)

	// Verify mapped_buffer_handle at offset 0 (8 bytes)
	handle := uint64(buf[0]) | uint64(buf[1])<<8 | uint64(buf[2])<<16 | uint64(buf[3])<<24 |
		uint64(buf[4])<<32 | uint64(buf[5])<<40 | uint64(buf[6])<<48 | uint64(buf[7])<<56
	if handle != 0x123456789ABCDEF0 {
		t.Errorf("mapped_buffer_handle = 0x%x, expected 0x123456789ABCDEF0", handle)
	}

	// Verify offset at offset 8 (4 bytes)
	offset := uint32(buf[8]) | uint32(buf[9])<<8 | uint32(buf[10])<<16 | uint32(buf[11])<<24
	if offset != 0x11223344 {
		t.Errorf("offset = 0x%x, expected 0x11223344", offset)
	}

	// Verify size at offset 12 (4 bytes)
	size := uint32(buf[12]) | uint32(buf[13])<<8 | uint32(buf[14])<<16 | uint32(buf[15])<<24
	if size != 0xDEADBEEF {
		t.Errorf("size = 0x%x, expected 0xDEADBEEF", size)
	}
}

// TestPackedDescListProgramParamsLayout verifies the struct layout
// Driver 4.20.0: 43 bytes, NO batch_size or stride fields
func TestPackedDescListProgramParamsLayout(t *testing.T) {
	params := NewPackedDescListProgramParams(
		0x1111111111111111, // buffer_handle
		0x2222222222222222, // buffer_size
		0x3333333333333333, // buffer_offset
		0x4444444444444444, // desc_handle
		0x55,               // channel_index
		0x66666666,         // starting_desc
		true,               // should_bind
		InterruptsDomainHost, // last_interrupts_domain
		true, // is_debug
	)

	// Total size should be 43 bytes
	if len(params) != 43 {
		t.Fatalf("params size = %d, expected 43", len(params))
	}

	// Verify buffer_handle at offset 0 (8 bytes)
	bufferHandle := uint64(params[0]) | uint64(params[1])<<8 | uint64(params[2])<<16 | uint64(params[3])<<24 |
		uint64(params[4])<<32 | uint64(params[5])<<40 | uint64(params[6])<<48 | uint64(params[7])<<56
	if bufferHandle != 0x1111111111111111 {
		t.Errorf("buffer_handle = 0x%x, expected 0x1111111111111111", bufferHandle)
	}

	// Verify channel_index at offset 32 (1 byte)
	if params[32] != 0x55 {
		t.Errorf("channel_index = 0x%x, expected 0x55", params[32])
	}

	// Verify should_bind at offset 37 (1 byte)
	if params[37] != 1 {
		t.Errorf("should_bind = %d, expected 1", params[37])
	}

	// Verify is_debug at offset 42 (1 byte)
	if params[42] != 1 {
		t.Errorf("is_debug = %d, expected 1", params[42])
	}
}

// TestPackedVdmaInterruptsWaitParamsLayout verifies the irq_data is 7 bytes per entry (driver 4.20.0)
func TestPackedVdmaInterruptsWaitParamsLayout(t *testing.T) {
	var bitmap [MaxVdmaEngines]uint32
	bitmap[0] = 0x12345678
	params := NewPackedVdmaInterruptsWaitParams(bitmap)

	// Total size should be 685 bytes: 12 (bitmap) + 1 (count) + 672 (96 * 7)
	if len(params) != 685 {
		t.Fatalf("params size = %d, expected 685", len(params))
	}

	// channels_count is at offset 12
	if params[12] != 0 {
		t.Errorf("channels_count should be 0 initially, got %d", params[12])
	}

	// Simulate kernel writing back irq_data (7 bytes per entry in 4.20.0)
	// Entry 0 at offset 13: engine_index, channel_index, is_active, transfers_completed, host_error, device_error, validation_success
	params[13] = 1     // engine_index
	params[14] = 5     // channel_index
	params[15] = 1     // is_active
	params[16] = 10    // transfers_completed
	params[17] = 0     // host_error
	params[18] = 0     // device_error
	params[19] = 1     // validation_success

	eng, ch, isActive, xfersCompleted, hostErr, devErr, validSuccess := params.IrqData(0)
	if eng != 1 || ch != 5 || !isActive || xfersCompleted != 10 || hostErr != 0 || devErr != 0 || !validSuccess {
		t.Errorf("IrqData(0) = (%d, %d, %v, %d, %d, %d, %v), expected (1, 5, true, 10, 0, 0, true)",
			eng, ch, isActive, xfersCompleted, hostErr, devErr, validSuccess)
	}

	// Entry 1 at offset 20 (13 + 7)
	params[20] = 2    // engine_index
	params[21] = 7    // channel_index
	params[22] = 0    // is_active (not active)
	params[23] = 0xff // transfers_completed = CHANNEL_NOT_ACTIVE
	params[24] = 0    // host_error
	params[25] = 0    // device_error
	params[26] = 0    // validation_success

	eng, ch, isActive, xfersCompleted, hostErr, devErr, validSuccess = params.IrqData(1)
	if eng != 2 || ch != 7 || isActive || xfersCompleted != HailoVdmaTransferDataChannelNotActive {
		t.Errorf("IrqData(1) = (%d, %d, %v, %d, ...), expected (2, 7, false, 0xff, ...)", eng, ch, isActive, xfersCompleted)
	}
}

// TestMaxBuffersPerSingleTransfer verifies the constant matches the driver 4.20.0
func TestMaxBuffersPerSingleTransfer(t *testing.T) {
	if MaxBuffersPerSingleTransfer != 2 {
		t.Errorf("MaxBuffersPerSingleTransfer = %d, expected 2 (HAILO_MAX_BUFFERS_PER_SINGLE_TRANSFER in 4.20.0)", MaxBuffersPerSingleTransfer)
	}
}
