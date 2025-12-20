package driver

// IOCTL Magic Values - must match hailo_ioctl_common.h
const (
	HailoGeneralIoctlMagic = 'g' // 0x67
	HailoVdmaIoctlMagic    = 'v' // 0x76
	HailoNncIoctlMagic     = 'n' // 0x6e
	HailoSocIoctlMagic     = 's' // 0x73
	HailoPciEpIoctlMagic   = 'p' // 0x70
)

// Driver Version - must match installed driver
const (
	HailoDrvVerMajor    = 4
	HailoDrvVerMinor    = 23
	HailoDrvVerRevision = 0
)

// VDMA Constants
const (
	MaxVdmaChannelsPerEngine             = 32
	VdmaChannelsPerEnginePerDirection    = 16
	MaxVdmaEngines                       = 3
	SizeOfVdmaDescriptor                 = 16
	VdmaDestChannelsStart                = 16
	MaxSgDescsCount                      = 64 * 1024
	HailoVdmaMaxOngoingTransfers         = 128
	ChannelIrqTimestampsSize             = HailoVdmaMaxOngoingTransfers * 2
	MaxControlLength                     = 1500
	HailoMaxBuffersPerSingleTransfer     = 8
	PcieExpectedMd5Length                = 16
	MaxFwLogBufferLength                 = 512
	MaxNotificationLength                = 1500
	HailoMaxStreamNameSize               = 64
	HailoMaxNetworkNameSize              = 64
	HailoMaxNetworkGroupNameSize         = 64
)

// Transfer data special values
const (
	HailoVdmaTransferDataChannelNotActive = 0xff
	HailoVdmaTransferDataChannelWithError = 0xfe
)

// BoardType represents the Hailo device type
type BoardType uint32

const (
	BoardTypeHailo8        BoardType = 0
	BoardTypeHailo15       BoardType = 1
	BoardTypeHailo15L      BoardType = 2
	BoardTypeHailo10H      BoardType = 3
	BoardTypeHailo10Legacy BoardType = 4
	BoardTypeMars          BoardType = 5
)

// DmaType represents the DMA interface type
type DmaType uint32

const (
	DmaTypePcie  DmaType = 0
	DmaTypeDram  DmaType = 1
	DmaTypePciEp DmaType = 2
)

// DmaDataDirection represents DMA transfer direction
type DmaDataDirection uint32

const (
	DmaBidirectional DmaDataDirection = 0
	DmaToDevice      DmaDataDirection = 1
	DmaFromDevice    DmaDataDirection = 2
	DmaNone          DmaDataDirection = 3
)

// DmaBufferType represents the type of DMA buffer
type DmaBufferType uint32

const (
	DmaUserPtrBuffer DmaBufferType = 0
	DmaDmabufBuffer  DmaBufferType = 1
)

// AllocationMode represents buffer allocation mode
type AllocationMode uint32

const (
	AllocationModeUserspace AllocationMode = 0
	AllocationModeDriver    AllocationMode = 1
)

// BufferSyncType represents sync direction
type BufferSyncType uint32

const (
	SyncForCpu    BufferSyncType = 0
	SyncForDevice BufferSyncType = 1
)

// InterruptsDomain represents interrupt notification domain
type InterruptsDomain uint32

const (
	InterruptsDomainNone   InterruptsDomain = 0
	InterruptsDomainDevice InterruptsDomain = 1 << 0
	InterruptsDomainHost   InterruptsDomain = 1 << 1
)

// CpuId represents the target CPU for firmware communication
type CpuId uint32

const (
	CpuIdCpu0 CpuId = 0
	CpuIdCpu1 CpuId = 1
	CpuIdNone CpuId = 2
)

// IOCTL command numbers - General
const (
	IoctlQueryDeviceProperties = 1
	IoctlQueryDriverInfo       = 2
)

// IOCTL command numbers - VDMA
const (
	IoctlVdmaEnableChannels          = 0
	IoctlVdmaDisableChannels         = 1
	IoctlVdmaInterruptsWait          = 2
	IoctlVdmaInterruptsReadTimestamp = 3
	IoctlVdmaBufferMap               = 4
	IoctlVdmaBufferUnmap             = 5
	IoctlVdmaBufferSync              = 6
	IoctlDescListCreate              = 7
	IoctlDescListRelease             = 8
	IoctlDescListProgram             = 9
	IoctlVdmaLowMemoryBufferAlloc    = 10
	IoctlVdmaLowMemoryBufferFree     = 11
	IoctlMarkAsInUse                 = 12
	IoctlVdmaContinuousBufferAlloc   = 13
	IoctlVdmaContinuousBufferFree    = 14
	IoctlVdmaLaunchTransfer          = 15
)

// IOCTL command numbers - NNC (Neural Network Core)
const (
	IoctlFwControl          = 0
	IoctlReadNotification   = 1
	IoctlDisableNotification = 2
	IoctlReadLog            = 3
	IoctlResetNnCore        = 4
	IoctlWriteActionList    = 5
)

// IOCTL command numbers - SOC
const (
	IoctlSocConnect  = 0
	IoctlSocClose    = 1
	IoctlSocPowerOff = 2
)

// IOCTL command numbers - PCI EP
const (
	IoctlPciEpAccept = 0
	IoctlPciEpClose  = 1
)

// IOCTL direction flags for _IOC macro
const (
	IocNone  = 0
	IocWrite = 1
	IocRead  = 2
)

// IOCTL size/direction encoding constants
const (
	IocNrBits   = 8
	IocTypeBits = 8
	IocSizeBits = 14
	IocDirBits  = 2

	IocNrShift   = 0
	IocTypeShift = IocNrShift + IocNrBits
	IocSizeShift = IocTypeShift + IocTypeBits
	IocDirShift  = IocSizeShift + IocSizeBits
)

// Ioc creates an IOCTL command number
func Ioc(dir, iocType, nr, size int) uint32 {
	return uint32((dir << IocDirShift) |
		(iocType << IocTypeShift) |
		(nr << IocNrShift) |
		(size << IocSizeShift))
}

// IoW creates a write IOCTL (data flows from user to kernel)
func IoW(iocType, nr, size int) uint32 {
	return Ioc(IocWrite, iocType, nr, size)
}

// IoR creates a read IOCTL (data flows from kernel to user)
func IoR(iocType, nr, size int) uint32 {
	return Ioc(IocRead, iocType, nr, size)
}

// IoWR creates a read-write IOCTL
func IoWR(iocType, nr, size int) uint32 {
	return Ioc(IocRead|IocWrite, iocType, nr, size)
}

// Io creates an IOCTL with no data transfer
func Io(iocType, nr int) uint32 {
	return Ioc(IocNone, iocType, nr, 0)
}
