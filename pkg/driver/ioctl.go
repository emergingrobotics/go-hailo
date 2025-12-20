package driver

import (
	"os"
	"unsafe"

	"golang.org/x/sys/unix"
)

// DeviceFile represents an open Hailo device file descriptor
type DeviceFile struct {
	fd   int
	path string
}

// OpenDevice opens a Hailo device by path
func OpenDevice(path string) (*DeviceFile, error) {
	fd, err := unix.Open(path, unix.O_RDWR|unix.O_CLOEXEC, 0)
	if err != nil {
		errno, ok := err.(unix.Errno)
		if ok {
			return nil, StatusFromErrno(errno, "opening device "+path)
		}
		return nil, NewErrorWithCause(StatusDriverOperationFailed, "opening device "+path, err)
	}
	return &DeviceFile{fd: fd, path: path}, nil
}

// Close closes the device file
func (d *DeviceFile) Close() error {
	if d.fd >= 0 {
		err := unix.Close(d.fd)
		d.fd = -1
		if err != nil {
			return NewErrorWithCause(StatusDriverOperationFailed, "closing device", err)
		}
	}
	return nil
}

// Fd returns the file descriptor
func (d *DeviceFile) Fd() int {
	return d.fd
}

// Path returns the device path
func (d *DeviceFile) Path() string {
	return d.path
}

// ioctl performs an ioctl syscall
func (d *DeviceFile) ioctl(cmd uint32, arg unsafe.Pointer) error {
	_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(d.fd), uintptr(cmd), uintptr(arg))
	if errno != 0 {
		return StatusFromErrno(errno, "ioctl")
	}
	return nil
}

// IOCTL command codes (calculated from type and size)
var (
	ioctlQueryDeviceProperties = IoW(int(HailoGeneralIoctlMagic), IoctlQueryDeviceProperties, SizeOfDeviceProperties)
	ioctlQueryDriverInfo       = IoW(int(HailoGeneralIoctlMagic), IoctlQueryDriverInfo, SizeOfDriverInfo)

	ioctlVdmaEnableChannels          = IoR(int(HailoVdmaIoctlMagic), IoctlVdmaEnableChannels, SizeOfVdmaEnableChannelsParams)
	ioctlVdmaDisableChannels         = IoR(int(HailoVdmaIoctlMagic), IoctlVdmaDisableChannels, SizeOfVdmaDisableChannelsParams)
	ioctlVdmaInterruptsWait          = IoWR(int(HailoVdmaIoctlMagic), IoctlVdmaInterruptsWait, SizeOfVdmaInterruptsWaitParams)
	ioctlVdmaBufferMap               = IoWR(int(HailoVdmaIoctlMagic), IoctlVdmaBufferMap, SizeOfVdmaBufferMapParams)
	ioctlVdmaBufferUnmap             = IoR(int(HailoVdmaIoctlMagic), IoctlVdmaBufferUnmap, SizeOfVdmaBufferUnmapParams)
	ioctlVdmaBufferSync              = IoR(int(HailoVdmaIoctlMagic), IoctlVdmaBufferSync, SizeOfVdmaBufferSyncParams)
	ioctlDescListCreate              = IoWR(int(HailoVdmaIoctlMagic), IoctlDescListCreate, SizeOfDescListCreateParams)
	ioctlDescListRelease             = IoR(int(HailoVdmaIoctlMagic), IoctlDescListRelease, SizeOfDescListReleaseParams)
	ioctlDescListProgram             = IoR(int(HailoVdmaIoctlMagic), IoctlDescListProgram, SizeOfDescListProgramParams)
	ioctlVdmaLaunchTransfer          = IoR(int(HailoVdmaIoctlMagic), IoctlVdmaLaunchTransfer, SizeOfVdmaLaunchTransferParams)

	ioctlFwControl       = IoWR(int(HailoNncIoctlMagic), IoctlFwControl, SizeOfFwControl)
	ioctlReadNotification = IoW(int(HailoNncIoctlMagic), IoctlReadNotification, SizeOfD2hNotification)
	ioctlResetNnCore     = Io(int(HailoNncIoctlMagic), IoctlResetNnCore)
)

// QueryDeviceProperties queries device properties via IOCTL
func (d *DeviceFile) QueryDeviceProperties() (*DeviceProperties, error) {
	var props DeviceProperties
	err := d.ioctl(ioctlQueryDeviceProperties, unsafe.Pointer(&props))
	if err != nil {
		return nil, err
	}
	return &props, nil
}

// QueryDriverInfo queries driver version information via IOCTL
func (d *DeviceFile) QueryDriverInfo() (*DriverInfo, error) {
	var info DriverInfo
	err := d.ioctl(ioctlQueryDriverInfo, unsafe.Pointer(&info))
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// VdmaEnableChannels enables VDMA channels
func (d *DeviceFile) VdmaEnableChannels(channelsBitmap [MaxVdmaEngines]uint32, enableTimestamps bool) error {
	params := VdmaEnableChannelsParams{
		ChannelsBitmapPerEngine: channelsBitmap,
		EnableTimestampsMeasure: enableTimestamps,
	}
	return d.ioctl(ioctlVdmaEnableChannels, unsafe.Pointer(&params))
}

// VdmaDisableChannels disables VDMA channels
func (d *DeviceFile) VdmaDisableChannels(channelsBitmap [MaxVdmaEngines]uint32) error {
	params := VdmaDisableChannelsParams{
		ChannelsBitmapPerEngine: channelsBitmap,
	}
	return d.ioctl(ioctlVdmaDisableChannels, unsafe.Pointer(&params))
}

// VdmaInterruptsWait waits for VDMA interrupts
func (d *DeviceFile) VdmaInterruptsWait(channelsBitmap [MaxVdmaEngines]uint32) (*VdmaInterruptsWaitParams, error) {
	params := VdmaInterruptsWaitParams{
		ChannelsBitmapPerEngine: channelsBitmap,
	}
	err := d.ioctl(ioctlVdmaInterruptsWait, unsafe.Pointer(&params))
	if err != nil {
		return nil, err
	}
	return &params, nil
}

// VdmaBufferMap maps a user buffer for DMA
func (d *DeviceFile) VdmaBufferMap(userAddr uintptr, size uint64, direction DmaDataDirection, bufferType DmaBufferType) (uint64, error) {
	params := VdmaBufferMapParams{
		UserAddress:           userAddr,
		Size:                  size,
		DataDirection:         direction,
		BufferType:            bufferType,
		AllocatedBufferHandle: ^uintptr(0), // -1 indicates no pre-allocated buffer
	}
	err := d.ioctl(ioctlVdmaBufferMap, unsafe.Pointer(&params))
	if err != nil {
		return 0, err
	}
	return params.MappedHandle, nil
}

// VdmaBufferUnmap unmaps a previously mapped buffer
func (d *DeviceFile) VdmaBufferUnmap(handle uint64) error {
	params := VdmaBufferUnmapParams{
		MappedHandle: handle,
	}
	return d.ioctl(ioctlVdmaBufferUnmap, unsafe.Pointer(&params))
}

// VdmaBufferSync synchronizes a buffer between CPU and device
func (d *DeviceFile) VdmaBufferSync(handle uint64, syncType BufferSyncType, offset, count uint64) error {
	params := VdmaBufferSyncParams{
		Handle:   handle,
		SyncType: syncType,
		Offset:   offset,
		Count:    count,
	}
	return d.ioctl(ioctlVdmaBufferSync, unsafe.Pointer(&params))
}

// DescListCreate creates a descriptor list
func (d *DeviceFile) DescListCreate(descCount uint64, pageSize uint16, isCircular bool) (uintptr, uint64, error) {
	params := DescListCreateParams{
		DescCount:    descCount,
		DescPageSize: pageSize,
		IsCircular:   isCircular,
	}
	err := d.ioctl(ioctlDescListCreate, unsafe.Pointer(&params))
	if err != nil {
		return 0, 0, err
	}
	return params.DescHandle, params.DmaAddress, nil
}

// DescListRelease releases a descriptor list
func (d *DeviceFile) DescListRelease(handle uintptr) error {
	params := DescListReleaseParams{
		DescHandle: handle,
	}
	return d.ioctl(ioctlDescListRelease, unsafe.Pointer(&params))
}

// FwControl sends a firmware control message
func (d *DeviceFile) FwControl(request []byte, md5 [16]byte, timeoutMs uint32, cpuId CpuId) ([]byte, [16]byte, error) {
	if len(request) > MaxControlLength {
		return nil, [16]byte{}, NewError(StatusInvalidArgument, "request too large")
	}

	params := FwControl{
		ExpectedMd5: md5,
		BufferLen:   uint32(len(request)),
		TimeoutMs:   timeoutMs,
		CpuId:       cpuId,
	}
	copy(params.Buffer[:], request)

	err := d.ioctl(ioctlFwControl, unsafe.Pointer(&params))
	if err != nil {
		return nil, [16]byte{}, err
	}

	response := make([]byte, params.BufferLen)
	copy(response, params.Buffer[:params.BufferLen])

	return response, params.ExpectedMd5, nil
}

// ResetNnCore resets the neural network core
func (d *DeviceFile) ResetNnCore() error {
	return d.ioctl(ioctlResetNnCore, nil)
}

// ReadNotification reads a device-to-host notification
func (d *DeviceFile) ReadNotification() ([]byte, error) {
	var params D2hNotification
	err := d.ioctl(ioctlReadNotification, unsafe.Pointer(&params))
	if err != nil {
		return nil, err
	}
	result := make([]byte, params.BufferLen)
	copy(result, params.Buffer[:params.BufferLen])
	return result, nil
}

// ScanDevices scans for available Hailo devices
func ScanDevices() ([]string, error) {
	// Check specific device paths /dev/hailo0 through /dev/hailo15
	var devices []string
	for i := 0; i < 16; i++ {
		path := "/dev/hailo" + string(rune('0'+i))
		if _, err := os.Stat(path); err == nil {
			devices = append(devices, path)
		}
	}
	return devices, nil
}
