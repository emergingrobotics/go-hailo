package driver

import (
	"encoding/binary"
	"unsafe"
)

// PackedDescListCreateParams is the packed version of DescListCreateParams
// matching C struct with #pragma pack(1): 27 bytes total
// struct hailo_desc_list_create_params {
//     size_t desc_count;          // 8 bytes, offset 0
//     uint16_t desc_page_size;    // 2 bytes, offset 8
//     bool is_circular;           // 1 byte,  offset 10
//     uintptr_t desc_handle;      // 8 bytes, offset 11 (output)
//     uint64_t dma_address;       // 8 bytes, offset 19 (output)
// };
type PackedDescListCreateParams [27]byte

func NewPackedDescListCreateParams(descCount uint64, descPageSize uint16, isCircular bool) *PackedDescListCreateParams {
	var p PackedDescListCreateParams
	binary.LittleEndian.PutUint64(p[0:8], descCount)
	binary.LittleEndian.PutUint16(p[8:10], descPageSize)
	if isCircular {
		p[10] = 1
	}
	return &p
}

func (p *PackedDescListCreateParams) DescHandle() uintptr {
	return uintptr(binary.LittleEndian.Uint64(p[11:19]))
}

func (p *PackedDescListCreateParams) DmaAddress() uint64 {
	return binary.LittleEndian.Uint64(p[19:27])
}

// PackedDescListReleaseParams: 8 bytes
// struct hailo_desc_list_release_params {
//     uintptr_t desc_handle;      // 8 bytes, offset 0
// };
type PackedDescListReleaseParams [8]byte

func NewPackedDescListReleaseParams(descHandle uintptr) *PackedDescListReleaseParams {
	var p PackedDescListReleaseParams
	binary.LittleEndian.PutUint64(p[0:8], uint64(descHandle))
	return &p
}

// PackedVdmaBufferMapParams: 48 bytes
// struct hailo_vdma_buffer_map_params {
//     uintptr_t user_address;                         // 8 bytes, offset 0
//     size_t size;                                    // 8 bytes, offset 8
//     enum hailo_dma_data_direction data_direction;   // 4 bytes, offset 16
//     enum hailo_dma_buffer_type buffer_type;         // 4 bytes, offset 20
//     uintptr_t allocated_buffer_handle;              // 8 bytes, offset 24
//     size_t mapped_handle;                           // 8 bytes, offset 32 (output)
// };
type PackedVdmaBufferMapParams [40]byte

func NewPackedVdmaBufferMapParams(userAddr uintptr, size uint64, direction DmaDataDirection, bufType DmaBufferType, allocHandle uintptr) *PackedVdmaBufferMapParams {
	var p PackedVdmaBufferMapParams
	binary.LittleEndian.PutUint64(p[0:8], uint64(userAddr))
	binary.LittleEndian.PutUint64(p[8:16], size)
	binary.LittleEndian.PutUint32(p[16:20], uint32(direction))
	binary.LittleEndian.PutUint32(p[20:24], uint32(bufType))
	binary.LittleEndian.PutUint64(p[24:32], uint64(allocHandle))
	return &p
}

func (p *PackedVdmaBufferMapParams) MappedHandle() uint64 {
	return binary.LittleEndian.Uint64(p[32:40])
}

// PackedVdmaBufferUnmapParams: 8 bytes
type PackedVdmaBufferUnmapParams [8]byte

func NewPackedVdmaBufferUnmapParams(handle uint64) *PackedVdmaBufferUnmapParams {
	var p PackedVdmaBufferUnmapParams
	binary.LittleEndian.PutUint64(p[0:8], handle)
	return &p
}

// PackedVdmaBufferSyncParams: 28 bytes
// struct hailo_vdma_buffer_sync_params {
//     size_t handle;                              // 8 bytes, offset 0
//     enum hailo_vdma_buffer_sync_type sync_type; // 4 bytes, offset 8
//     size_t offset;                              // 8 bytes, offset 12
//     size_t count;                               // 8 bytes, offset 20
// };
type PackedVdmaBufferSyncParams [28]byte

func NewPackedVdmaBufferSyncParams(handle uint64, syncType BufferSyncType, offset, count uint64) *PackedVdmaBufferSyncParams {
	var p PackedVdmaBufferSyncParams
	binary.LittleEndian.PutUint64(p[0:8], handle)
	binary.LittleEndian.PutUint32(p[8:12], uint32(syncType))
	binary.LittleEndian.PutUint64(p[12:20], offset)
	binary.LittleEndian.PutUint64(p[20:28], count)
	return &p
}

// PackedVdmaEnableChannelsParams: 16 bytes
// struct hailo_vdma_enable_channels_params {
//     uint32_t channels_bitmap_per_engine[MAX_VDMA_ENGINES_COUNT]; // 12 bytes
//     bool enable_timestamps_measure;                               // 1 byte
// };
type PackedVdmaEnableChannelsParams [13]byte

func NewPackedVdmaEnableChannelsParams(bitmap [MaxVdmaEngines]uint32, enableTimestamps bool) *PackedVdmaEnableChannelsParams {
	var p PackedVdmaEnableChannelsParams
	for i := 0; i < MaxVdmaEngines; i++ {
		binary.LittleEndian.PutUint32(p[i*4:(i+1)*4], bitmap[i])
	}
	if enableTimestamps {
		p[12] = 1
	}
	return &p
}

// PackedVdmaDisableChannelsParams: 12 bytes
type PackedVdmaDisableChannelsParams [12]byte

func NewPackedVdmaDisableChannelsParams(bitmap [MaxVdmaEngines]uint32) *PackedVdmaDisableChannelsParams {
	var p PackedVdmaDisableChannelsParams
	for i := 0; i < MaxVdmaEngines; i++ {
		binary.LittleEndian.PutUint32(p[i*4:(i+1)*4], bitmap[i])
	}
	return &p
}

// PackedVdmaInterruptsWaitParams: 685 bytes
// struct hailo_vdma_interrupts_wait_params {
//     uint32_t channels_bitmap_per_engine[MAX_VDMA_ENGINES]; // 12 bytes, offset 0
//     uint8_t channels_count;                                 // 1 byte, offset 12 (output)
//     struct hailo_vdma_interrupts_channel_data irq_data[96]; // 672 bytes, offset 13
// };
// hailo_vdma_interrupts_channel_data is 7 bytes:
//   engine_index(1) + channel_index(1) + is_active(1) + transfers_completed(1) +
//   host_error(1) + device_error(1) + validation_success(1)
const PackedVdmaInterruptsWaitParamsSize = 12 + 1 + (MaxVdmaChannelsPerEngine * MaxVdmaEngines * 7)

type PackedVdmaInterruptsWaitParams [685]byte

func NewPackedVdmaInterruptsWaitParams(bitmap [MaxVdmaEngines]uint32) *PackedVdmaInterruptsWaitParams {
	var p PackedVdmaInterruptsWaitParams
	for i := 0; i < MaxVdmaEngines; i++ {
		binary.LittleEndian.PutUint32(p[i*4:(i+1)*4], bitmap[i])
	}
	return &p
}

func (p *PackedVdmaInterruptsWaitParams) ChannelsCount() uint8 {
	return p[12]
}

// IrqData returns the interrupt data for a channel at the given index
func (p *PackedVdmaInterruptsWaitParams) IrqData(idx int) (engineIndex, channelIndex uint8, isActive bool, transfersCompleted, hostError, deviceError uint8, validationSuccess bool) {
	offset := 13 + idx*7
	return p[offset], p[offset+1], p[offset+2] != 0, p[offset+3], p[offset+4], p[offset+5], p[offset+6] != 0
}

// PackedDescListProgramParams: 43 bytes
// struct hailo_desc_list_program_params {
//     size_t buffer_handle;                                    // 8 bytes, offset 0
//     size_t buffer_size;                                      // 8 bytes, offset 8
//     size_t buffer_offset;                                    // 8 bytes, offset 16
//     uintptr_t desc_handle;                                   // 8 bytes, offset 24
//     uint8_t channel_index;                                   // 1 byte,  offset 32
//     uint32_t starting_desc;                                  // 4 bytes, offset 33
//     bool should_bind;                                        // 1 byte,  offset 37
//     enum hailo_vdma_interrupts_domain last_interrupts_domain;// 4 bytes, offset 38
//     bool is_debug;                                           // 1 byte,  offset 42
// };
type PackedDescListProgramParams [43]byte

func NewPackedDescListProgramParams(bufferHandle, bufferSize, bufferOffset uint64, descHandle uintptr, channelIndex uint8, startingDesc uint32, shouldBind bool, lastInterruptsDomain InterruptsDomain, isDebug bool) *PackedDescListProgramParams {
	var p PackedDescListProgramParams
	binary.LittleEndian.PutUint64(p[0:8], bufferHandle)
	binary.LittleEndian.PutUint64(p[8:16], bufferSize)
	binary.LittleEndian.PutUint64(p[16:24], bufferOffset)
	binary.LittleEndian.PutUint64(p[24:32], uint64(descHandle))
	p[32] = channelIndex
	binary.LittleEndian.PutUint32(p[33:37], startingDesc)
	if shouldBind {
		p[37] = 1
	}
	binary.LittleEndian.PutUint32(p[38:42], uint32(lastInterruptsDomain))
	if isDebug {
		p[42] = 1
	}
	return &p
}

// MaxBuffersPerSingleTransfer matches HAILO_MAX_BUFFERS_PER_SINGLE_TRANSFER
const MaxBuffersPerSingleTransfer = 2

// PackedVdmaTransferBuffer: 16 bytes
// struct hailo_vdma_transfer_buffer {
//     size_t mapped_buffer_handle;  // 8 bytes, offset 0
//     uint32_t offset;              // 4 bytes, offset 8
//     uint32_t size;                // 4 bytes, offset 12
// };
type PackedVdmaTransferBuffer [16]byte

func NewPackedVdmaTransferBuffer(mappedHandle uint64, offset, size uint32) *PackedVdmaTransferBuffer {
	var p PackedVdmaTransferBuffer
	binary.LittleEndian.PutUint64(p[0:8], mappedHandle)
	binary.LittleEndian.PutUint32(p[8:12], offset)
	binary.LittleEndian.PutUint32(p[12:16], size)
	return &p
}

// PackedVdmaLaunchTransferParams: 65 bytes
// struct hailo_vdma_launch_transfer_params {
//     uint8_t engine_index;                                    // 1 byte,  offset 0
//     uint8_t channel_index;                                   // 1 byte,  offset 1
//     uintptr_t desc_handle;                                   // 8 bytes, offset 2
//     uint32_t starting_desc;                                  // 4 bytes, offset 10
//     bool should_bind;                                        // 1 byte,  offset 14
//     uint8_t buffers_count;                                   // 1 byte,  offset 15
//     struct hailo_vdma_transfer_buffer buffers[2];            // 32 bytes, offset 16
//     enum hailo_vdma_interrupts_domain first_interrupts_domain;// 4 bytes, offset 48
//     enum hailo_vdma_interrupts_domain last_interrupts_domain; // 4 bytes, offset 52
//     bool is_debug;                                           // 1 byte,  offset 56
//     uint32_t descs_programed;                                // 4 bytes, offset 57 (output)
//     int launch_transfer_status;                              // 4 bytes, offset 61 (output)
// };
type PackedVdmaLaunchTransferParams [65]byte

func NewPackedVdmaLaunchTransferParams(engineIndex, channelIndex uint8, descHandle uintptr, startingDesc uint32, shouldBind bool, buffers []PackedVdmaTransferBuffer, firstDomain, lastDomain InterruptsDomain, isDebug bool) *PackedVdmaLaunchTransferParams {
	var p PackedVdmaLaunchTransferParams
	p[0] = engineIndex
	p[1] = channelIndex
	binary.LittleEndian.PutUint64(p[2:10], uint64(descHandle))
	binary.LittleEndian.PutUint32(p[10:14], startingDesc)
	if shouldBind {
		p[14] = 1
	}
	p[15] = uint8(len(buffers))
	for i, buf := range buffers {
		if i >= MaxBuffersPerSingleTransfer {
			break
		}
		copy(p[16+i*16:16+(i+1)*16], buf[:])
	}
	binary.LittleEndian.PutUint32(p[48:52], uint32(firstDomain))
	binary.LittleEndian.PutUint32(p[52:56], uint32(lastDomain))
	if isDebug {
		p[56] = 1
	}
	return &p
}

func (p *PackedVdmaLaunchTransferParams) DescsProgramed() uint32 {
	return binary.LittleEndian.Uint32(p[57:61])
}

func (p *PackedVdmaLaunchTransferParams) LaunchTransferStatus() int32 {
	return int32(binary.LittleEndian.Uint32(p[61:65]))
}

// Packed size constants for ioctl commands
const (
	SizeOfPackedDescListCreateParams       = int(unsafe.Sizeof(PackedDescListCreateParams{}))
	SizeOfPackedDescListReleaseParams      = int(unsafe.Sizeof(PackedDescListReleaseParams{}))
	SizeOfPackedDescListProgramParams      = int(unsafe.Sizeof(PackedDescListProgramParams{}))
	SizeOfPackedVdmaBufferMapParams        = int(unsafe.Sizeof(PackedVdmaBufferMapParams{}))
	SizeOfPackedVdmaBufferUnmapParams      = int(unsafe.Sizeof(PackedVdmaBufferUnmapParams{}))
	SizeOfPackedVdmaBufferSyncParams       = int(unsafe.Sizeof(PackedVdmaBufferSyncParams{}))
	SizeOfPackedVdmaEnableChannelsParams   = int(unsafe.Sizeof(PackedVdmaEnableChannelsParams{}))
	SizeOfPackedVdmaDisableChannelsParams  = int(unsafe.Sizeof(PackedVdmaDisableChannelsParams{}))
	SizeOfPackedVdmaInterruptsWaitParams   = int(unsafe.Sizeof(PackedVdmaInterruptsWaitParams{}))
	SizeOfPackedVdmaLaunchTransferParams   = int(unsafe.Sizeof(PackedVdmaLaunchTransferParams{}))
)
