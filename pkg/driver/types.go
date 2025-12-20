package driver

import "unsafe"

// DeviceProperties matches struct hailo_device_properties from driver
type DeviceProperties struct {
	DescMaxPageSize uint16
	_               [2]byte // padding
	BoardType       BoardType
	AllocationMode  AllocationMode
	DmaType         DmaType
	DmaEnginesCount uint64
	IsFwLoaded      bool
	_               [7]byte // padding to 32 bytes
}

// DriverInfo matches struct hailo_driver_info from driver
type DriverInfo struct {
	MajorVersion    uint32
	MinorVersion    uint32
	RevisionVersion uint32
}

// VdmaBufferMapParams matches struct hailo_vdma_buffer_map_params
type VdmaBufferMapParams struct {
	UserAddress           uintptr
	Size                  uint64
	DataDirection         DmaDataDirection
	BufferType            DmaBufferType
	AllocatedBufferHandle uintptr
	MappedHandle          uint64 // output
}

// VdmaBufferUnmapParams matches struct hailo_vdma_buffer_unmap_params
type VdmaBufferUnmapParams struct {
	MappedHandle uint64
}

// VdmaBufferSyncParams matches struct hailo_vdma_buffer_sync_params
type VdmaBufferSyncParams struct {
	Handle   uint64
	SyncType BufferSyncType
	Offset   uint64
	Count    uint64
}

// DescListCreateParams matches struct hailo_desc_list_create_params
type DescListCreateParams struct {
	DescCount    uint64
	DescPageSize uint16
	IsCircular   bool
	_            [5]byte // padding
	DescHandle   uintptr // output
	DmaAddress   uint64  // output
}

// DescListReleaseParams matches struct hailo_desc_list_release_params
type DescListReleaseParams struct {
	DescHandle uintptr
}

// DescListProgramParams matches struct hailo_desc_list_program_params
type DescListProgramParams struct {
	BufferHandle         uint64
	BufferSize           uint64
	BufferOffset         uint64
	BatchSize            uint32
	_                    [4]byte // padding
	DescHandle           uintptr
	ChannelIndex         uint8
	_                    [3]byte // padding
	StartingDesc         uint32
	ShouldBind           bool
	_                    [3]byte // padding
	LastInterruptsDomain InterruptsDomain
	IsDebug              bool
	_                    [3]byte // padding
	Stride               uint32
}

// VdmaEnableChannelsParams matches struct hailo_vdma_enable_channels_params
type VdmaEnableChannelsParams struct {
	ChannelsBitmapPerEngine  [MaxVdmaEngines]uint32
	EnableTimestampsMeasure bool
	_                       [3]byte // padding
}

// VdmaDisableChannelsParams matches struct hailo_vdma_disable_channels_params
type VdmaDisableChannelsParams struct {
	ChannelsBitmapPerEngine [MaxVdmaEngines]uint32
}

// VdmaInterruptsChannelData matches struct hailo_vdma_interrupts_channel_data
type VdmaInterruptsChannelData struct {
	EngineIndex  uint8
	ChannelIndex uint8
	Data         uint8
	_            uint8 // padding
}

// VdmaInterruptsWaitParams matches struct hailo_vdma_interrupts_wait_params
type VdmaInterruptsWaitParams struct {
	ChannelsBitmapPerEngine [MaxVdmaEngines]uint32
	ChannelsCount           uint8
	_                       [3]byte // padding
	IrqData                 [MaxVdmaChannelsPerEngine * MaxVdmaEngines]VdmaInterruptsChannelData
}

// ChannelInterruptTimestamp matches struct hailo_channel_interrupt_timestamp
type ChannelInterruptTimestamp struct {
	TimestampNs      uint64
	DescNumProcessed uint16
	_                [6]byte // padding
}

// VdmaInterruptsReadTimestampParams matches struct hailo_vdma_interrupts_read_timestamp_params
type VdmaInterruptsReadTimestampParams struct {
	EngineIndex     uint8
	ChannelIndex    uint8
	_               [2]byte // padding
	TimestampsCount uint32  // output
	Timestamps      [ChannelIrqTimestampsSize]ChannelInterruptTimestamp
}

// VdmaTransferBuffer matches struct hailo_vdma_transfer_buffer
type VdmaTransferBuffer struct {
	BufferType DmaBufferType
	_          [4]byte // padding
	AddrOrFd   uintptr
	Size       uint32
	_          [4]byte // padding
}

// VdmaLaunchTransferParams matches struct hailo_vdma_launch_transfer_params
type VdmaLaunchTransferParams struct {
	EngineIndex            uint8
	ChannelIndex           uint8
	_                      [6]byte // padding
	DescHandle             uintptr
	StartingDesc           uint32
	ShouldBind             bool
	BuffersCount           uint8
	_                      [2]byte // padding
	Buffers                [HailoMaxBuffersPerSingleTransfer]VdmaTransferBuffer
	FirstInterruptsDomain  InterruptsDomain
	LastInterruptsDomain   InterruptsDomain
	IsDebug                bool
	_                      [3]byte // padding
}

// FwControl matches struct hailo_fw_control
type FwControl struct {
	ExpectedMd5 [PcieExpectedMd5Length]byte
	BufferLen   uint32
	Buffer      [MaxControlLength]byte
	TimeoutMs   uint32
	CpuId       CpuId
}

// D2hNotification matches struct hailo_d2h_notification
type D2hNotification struct {
	BufferLen uint64
	Buffer    [MaxNotificationLength]byte
}

// ReadLogParams matches struct hailo_read_log_params
type ReadLogParams struct {
	CpuId      CpuId
	_          [4]byte // padding
	Buffer     [MaxFwLogBufferLength]byte
	BufferSize uint64
	ReadBytes  uint64 // output
}

// WriteActionListParams matches struct hailo_write_action_list_params
type WriteActionListParams struct {
	Data       uintptr
	Size       uint64
	DmaAddress uint64 // output
}

// AllocateContinuousBufferParams matches struct hailo_allocate_continuous_buffer_params
type AllocateContinuousBufferParams struct {
	BufferSize   uint64
	BufferHandle uintptr // output
	DmaAddress   uint64  // output
}

// FreeContinuousBufferParams matches struct hailo_free_continuous_buffer_params
type FreeContinuousBufferParams struct {
	BufferHandle uintptr
}

// AllocateLowMemoryBufferParams matches struct hailo_allocate_low_memory_buffer_params
type AllocateLowMemoryBufferParams struct {
	BufferSize   uint64
	BufferHandle uintptr // output
}

// FreeLowMemoryBufferParams matches struct hailo_free_low_memory_buffer_params
type FreeLowMemoryBufferParams struct {
	BufferHandle uintptr
}

// MarkAsInUseParams matches struct hailo_mark_as_in_use_params
type MarkAsInUseParams struct {
	InUse bool
	_     [7]byte // padding
}

// SocConnectParams matches struct hailo_soc_connect_params
type SocConnectParams struct {
	PortNumber         uint16
	InputChannelIndex  uint8  // output
	OutputChannelIndex uint8  // output
	_                  [4]byte // padding
	InputDescHandle    uintptr
	OutputDescHandle   uintptr
}

// SocCloseParams matches struct hailo_soc_close_params
type SocCloseParams struct {
	InputChannelIndex  uint8
	OutputChannelIndex uint8
	_                  [6]byte // padding
}

// PciEpAcceptParams matches struct hailo_pci_ep_accept_params
type PciEpAcceptParams struct {
	PortNumber         uint16
	InputChannelIndex  uint8  // output
	OutputChannelIndex uint8  // output
	_                  [4]byte // padding
	InputDescHandle    uintptr
	OutputDescHandle   uintptr
}

// PciEpCloseParams matches struct hailo_pci_ep_close_params
type PciEpCloseParams struct {
	InputChannelIndex  uint8
	OutputChannelIndex uint8
	_                  [6]byte // padding
}

// Size constants for struct validation
const (
	SizeOfDeviceProperties               = int(unsafe.Sizeof(DeviceProperties{}))
	SizeOfDriverInfo                     = int(unsafe.Sizeof(DriverInfo{}))
	SizeOfVdmaBufferMapParams            = int(unsafe.Sizeof(VdmaBufferMapParams{}))
	SizeOfVdmaBufferUnmapParams          = int(unsafe.Sizeof(VdmaBufferUnmapParams{}))
	SizeOfVdmaBufferSyncParams           = int(unsafe.Sizeof(VdmaBufferSyncParams{}))
	SizeOfDescListCreateParams           = int(unsafe.Sizeof(DescListCreateParams{}))
	SizeOfDescListReleaseParams          = int(unsafe.Sizeof(DescListReleaseParams{}))
	SizeOfDescListProgramParams          = int(unsafe.Sizeof(DescListProgramParams{}))
	SizeOfVdmaEnableChannelsParams       = int(unsafe.Sizeof(VdmaEnableChannelsParams{}))
	SizeOfVdmaDisableChannelsParams      = int(unsafe.Sizeof(VdmaDisableChannelsParams{}))
	SizeOfVdmaInterruptsWaitParams       = int(unsafe.Sizeof(VdmaInterruptsWaitParams{}))
	SizeOfVdmaLaunchTransferParams       = int(unsafe.Sizeof(VdmaLaunchTransferParams{}))
	SizeOfFwControl                      = int(unsafe.Sizeof(FwControl{}))
	SizeOfD2hNotification                = int(unsafe.Sizeof(D2hNotification{}))
	SizeOfChannelInterruptTimestamp      = int(unsafe.Sizeof(ChannelInterruptTimestamp{}))
	SizeOfVdmaTransferBuffer             = int(unsafe.Sizeof(VdmaTransferBuffer{}))
)
