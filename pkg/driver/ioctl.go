package driver

import (
	"context"
	"fmt"
	"os"
	"time"
	"unsafe"

	"golang.org/x/sys/unix"
)

// DefaultIoctlTimeout is the default timeout for ioctl operations
const DefaultIoctlTimeout = 5 * time.Second

// DeviceFile represents an open Hailo device file descriptor
type DeviceFile struct {
	fd   int
	path string
}

// OpenDevice opens a Hailo device by path
func OpenDevice(path string) (*DeviceFile, error) {
	return OpenDeviceWithTimeout(path, DefaultIoctlTimeout)
}

// OpenDeviceWithTimeout opens a Hailo device with a timeout
func OpenDeviceWithTimeout(path string, timeout time.Duration) (*DeviceFile, error) {
	type result struct {
		fd  int
		err error
	}

	done := make(chan result, 1)

	go func() {
		// Note: Don't use O_NONBLOCK as it interferes with the Hailo driver's
		// semaphore acquisition (causes "down_interruptible fail" errors)
		fd, err := unix.Open(path, unix.O_RDWR|unix.O_CLOEXEC, 0)
		done <- result{fd, err}
	}()

	select {
	case r := <-done:
		if r.err != nil {
			errno, ok := r.err.(unix.Errno)
			if ok {
				return nil, StatusFromErrno(errno, "opening device "+path)
			}
			return nil, NewErrorWithCause(StatusDriverOperationFailed, "opening device "+path, r.err)
		}
		return &DeviceFile{fd: r.fd, path: path}, nil
	case <-time.After(timeout):
		return nil, NewError(StatusTimeout, fmt.Sprintf("opening device %s timed out after %v (device may be locked by another process)", path, timeout))
	}
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

// ioctlWithTimeout performs an ioctl with a timeout.
// If the ioctl doesn't complete within the timeout, it returns ErrTimeout.
// Note: The underlying ioctl may still be running in the background.
func (d *DeviceFile) ioctlWithTimeout(ctx context.Context, cmd uint32, arg unsafe.Pointer, timeout time.Duration) error {
	if timeout == 0 {
		timeout = DefaultIoctlTimeout
	}

	// Create a context with timeout if not already set
	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), timeout)
		defer cancel()
	}

	// Channel to receive the result
	done := make(chan error, 1)

	go func() {
		_, _, errno := unix.Syscall(unix.SYS_IOCTL, uintptr(d.fd), uintptr(cmd), uintptr(arg))
		if errno != 0 {
			done <- StatusFromErrno(errno, "ioctl")
		} else {
			done <- nil
		}
	}()

	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return NewError(StatusTimeout, fmt.Sprintf("ioctl 0x%08x timed out after %v", cmd, timeout))
	}
}

// IOCTL command codes (calculated from type and size)
// Note: Hailo driver uses _IOW_ for query operations (counterintuitive but matches driver source)
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
// Note: Some driver versions (e.g., RPi 4.20.0) don't implement this IOCTL.
// In that case, we return sensible defaults for Hailo-8.
func (d *DeviceFile) QueryDeviceProperties() (*DeviceProperties, error) {
	var props DeviceProperties
	err := d.ioctlWithTimeout(nil, ioctlQueryDeviceProperties, unsafe.Pointer(&props), DefaultIoctlTimeout)
	if err != nil {
		// If IOCTL not supported or timed out, return default Hailo-8 properties
		hailoErr, ok := err.(*HailoError)
		if ok && (hailoErr.Status == StatusDriverInvalidIoctl || hailoErr.Status == StatusTimeout) {
			return &DeviceProperties{
				DescMaxPageSize: 4096,
				BoardType:       BoardTypeHailo8,
				AllocationMode:  AllocationModeUserspace,
				DmaType:         DmaTypePcie,
				DmaEnginesCount: 3,
				IsFwLoaded:      true,
			}, nil
		}
		return nil, err
	}
	return &props, nil
}

// QueryDriverInfo queries driver version information via IOCTL
func (d *DeviceFile) QueryDriverInfo() (*DriverInfo, error) {
	var info DriverInfo
	err := d.ioctlWithTimeout(nil, ioctlQueryDriverInfo, unsafe.Pointer(&info), DefaultIoctlTimeout)
	if err != nil {
		// If IOCTL times out, return a default version
		hailoErr, ok := err.(*HailoError)
		if ok && hailoErr.Status == StatusTimeout {
			return &DriverInfo{
				MajorVersion:    4,
				MinorVersion:    20,
				RevisionVersion: 0,
			}, nil
		}
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

// InferenceTimeout is the timeout for inference operations (longer than default)
const InferenceTimeout = 30 * time.Second

// VdmaInterruptsWait waits for VDMA interrupts
func (d *DeviceFile) VdmaInterruptsWait(channelsBitmap [MaxVdmaEngines]uint32) (*VdmaInterruptsWaitParams, error) {
	return d.VdmaInterruptsWaitWithTimeout(channelsBitmap, InferenceTimeout)
}

// VdmaInterruptsWaitWithTimeout waits for VDMA interrupts with a custom timeout
func (d *DeviceFile) VdmaInterruptsWaitWithTimeout(channelsBitmap [MaxVdmaEngines]uint32, timeout time.Duration) (*VdmaInterruptsWaitParams, error) {
	params := VdmaInterruptsWaitParams{
		ChannelsBitmapPerEngine: channelsBitmap,
	}
	err := d.ioctlWithTimeout(nil, ioctlVdmaInterruptsWait, unsafe.Pointer(&params), timeout)
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

// DescListProgram programs a descriptor list with buffer information
func (d *DeviceFile) DescListProgram(params *DescListProgramParams) error {
	return d.ioctl(ioctlDescListProgram, unsafe.Pointer(params))
}

// VdmaLaunchTransfer launches a VDMA transfer
func (d *DeviceFile) VdmaLaunchTransfer(params *VdmaLaunchTransferParams) error {
	return d.ioctl(ioctlVdmaLaunchTransfer, unsafe.Pointer(params))
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
		path := fmt.Sprintf("/dev/hailo%d", i)
		if _, err := os.Stat(path); err == nil {
			devices = append(devices, path)
		}
	}
	return devices, nil
}

// GetIoctlQueryDeviceProperties returns the IOCTL command code for debugging
func GetIoctlQueryDeviceProperties() uint32 {
	return ioctlQueryDeviceProperties
}

// GetIoctlQueryDriverInfo returns the IOCTL command code for debugging
func GetIoctlQueryDriverInfo() uint32 {
	return ioctlQueryDriverInfo
}
