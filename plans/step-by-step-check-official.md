# Step-by-Step Guide: Debugging SET_NETWORK_GROUP_HEADER Firmware Error

## Problem Summary

The Go implementation of `SET_NETWORK_GROUP_HEADER` returns firmware error `0x40030060` (CONTROL_PROTOCOL_STATUS_INVALID_CONTEXT_SWITCH_APP_HEADER_LENGTH). The struct format has been verified to match the SDK's 32-byte definition, but the Hailo8L firmware rejects it.

This guide provides steps to diagnose and fix the issue.

---

## Step 1: Install HailoRT SDK on Raspberry Pi 5

### 1.1 Check Current System State

```bash
# Check if any Hailo packages are installed
dpkg -l | grep -i hailo

# Check kernel module
cat /proc/modules | grep hailo

# Check device exists
ls -la /dev/hailo*

# Check driver version
cat /sys/class/hailo_chardev/hailo0/device_id
```

### 1.2 Add Hailo Repository

```bash
# Add Hailo's GPG key
wget -qO - https://hailo-hrt.github.io/hailo_ai_sw_suite/key.gpg | sudo apt-key add -

# Add repository (for Raspberry Pi OS / Debian)
echo "deb https://hailo-hrt.github.io/hailo_ai_sw_suite/rpi5 ./" | sudo tee /etc/apt/sources.list.d/hailo.list

# Update package list
sudo apt update
```

### 1.3 Install HailoRT Packages

```bash
# Install the HailoRT runtime library and tools
sudo apt install hailort

# Install Python bindings
sudo apt install hailort-python3

# Or if using pip:
pip3 install hailort

# Verify installation
hailortcli --version
python3 -c "import hailo_platform; print(hailo_platform.__version__)"
```

### 1.4 Alternative: Manual Installation from Hailo Developer Zone

If the repository doesn't work:

1. Go to https://hailo.ai/developer-zone/
2. Create an account and log in
3. Download "HailoRT" for your platform (ARM64 / Raspberry Pi)
4. Download the `.deb` packages or wheel files
5. Install manually:

```bash
# For .deb packages
sudo dpkg -i hailort_*.deb
sudo dpkg -i hailort-python3_*.deb

# For Python wheel
pip3 install hailort-*.whl
```

---

## Step 2: Verify Device Works with Official Examples

### 2.1 Basic Device Check with hailortcli

```bash
# Scan for devices
hailortcli scan

# Get device info
hailortcli fw-control identify

# Expected output should show firmware version, board name, etc.
```

### 2.2 Run Python Inference Example

Create a test script `/tmp/test_hailo.py`:

```python
#!/usr/bin/env python3
"""
Minimal test to verify Hailo device works with official SDK.
"""
import sys
from hailo_platform import VDevice, HEF, ConfigureParams, InferVStreams, InputVStreamParams, OutputVStreamParams
import numpy as np

def main():
    hef_path = sys.argv[1] if len(sys.argv) > 1 else "/go-hailo/models/yolox_s_leaky_hailo8.hef"

    print(f"Loading HEF: {hef_path}")

    # Create virtual device (auto-selects available device)
    with VDevice() as vdevice:
        print(f"Device created: {vdevice}")

        # Load HEF
        hef = HEF(hef_path)
        print(f"HEF loaded: {hef.get_network_group_names()}")

        # Configure network group
        configure_params = ConfigureParams.create_from_hef(hef, interface=vdevice.get_default_interface())
        network_group = vdevice.configure(hef, configure_params)[0]
        print(f"Network group configured: {network_group.name}")

        # Get input/output info
        input_vstreams_params = InputVStreamParams.make(network_group)
        output_vstreams_params = OutputVStreamParams.make(network_group)

        print("Input streams:")
        for name, params in input_vstreams_params.items():
            print(f"  {name}: shape={params.shape}, format={params.format}")

        print("Output streams:")
        for name, params in output_vstreams_params.items():
            print(f"  {name}: shape={params.shape}, format={params.format}")

        # Create dummy input and run inference
        with InferVStreams(network_group, input_vstreams_params, output_vstreams_params) as infer:
            input_data = {}
            for name, params in input_vstreams_params.items():
                # Create random input matching expected shape
                shape = params.shape
                input_data[name] = np.random.randint(0, 255, shape, dtype=np.uint8)
                print(f"Created input for {name}: shape={shape}")

            print("Running inference...")
            outputs = infer.infer(input_data)

            print("Inference complete! Output shapes:")
            for name, data in outputs.items():
                print(f"  {name}: {data.shape}")

        print("\nSUCCESS: Device works correctly with official SDK!")

if __name__ == "__main__":
    main()
```

Run the test:

```bash
python3 /tmp/test_hailo.py /go-hailo/models/yolox_s_leaky_hailo8.hef
```

**If this works**: The device and firmware are functioning correctly, and the issue is in our Go implementation.

**If this fails**: There may be a driver/firmware issue that needs to be resolved first.

---

## Step 3: Trace SDK's Actual Wire Protocol

This is the critical step - we need to see exactly what bytes the SDK sends to the device.

### 3.1 Method A: Using LD_PRELOAD to Intercept ioctl

Create an ioctl interceptor `/tmp/ioctl_trace.c`:

```c
#define _GNU_SOURCE
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>
#include <dlfcn.h>
#include <sys/ioctl.h>
#include <fcntl.h>
#include <unistd.h>

// Hailo ioctl magic numbers
#define HAILO_NNC_IOCTL_MAGIC 'H'
#define HAILO_FW_CONTROL_NR 0  // FW_CONTROL is ioctl number 0

// FwControl structure (must match driver)
struct hailo_fw_control {
    uint8_t expected_md5[16];
    uint32_t buffer_len;
    uint8_t buffer[1500];
    uint32_t timeout_ms;
    uint32_t cpu_id;
};

static int (*real_ioctl)(int fd, unsigned long request, ...) = NULL;

static void print_hex(const char* label, const uint8_t* data, size_t len) {
    fprintf(stderr, "[TRACE] %s (%zu bytes):\n", label, len);
    for (size_t i = 0; i < len && i < 128; i++) {
        fprintf(stderr, "%02x ", data[i]);
        if ((i + 1) % 16 == 0) fprintf(stderr, "\n");
    }
    if (len > 128) fprintf(stderr, "... (truncated)\n");
    else if (len % 16 != 0) fprintf(stderr, "\n");
}

int ioctl(int fd, unsigned long request, ...) {
    if (!real_ioctl) {
        real_ioctl = dlsym(RTLD_NEXT, "ioctl");
    }

    va_list args;
    va_start(args, request);
    void* arg = va_arg(args, void*);
    va_end(args);

    // Check if this is a Hailo NNC ioctl
    if (_IOC_TYPE(request) == HAILO_NNC_IOCTL_MAGIC) {
        int nr = _IOC_NR(request);
        fprintf(stderr, "\n[TRACE] === Hailo ioctl: nr=%d, size=%lu ===\n",
                nr, (unsigned long)_IOC_SIZE(request));

        if (nr == HAILO_FW_CONTROL_NR && arg != NULL) {
            struct hailo_fw_control* ctrl = (struct hailo_fw_control*)arg;
            fprintf(stderr, "[TRACE] FW_CONTROL: buffer_len=%u, timeout=%u, cpu_id=%u\n",
                    ctrl->buffer_len, ctrl->timeout_ms, ctrl->cpu_id);
            print_hex("Request buffer", ctrl->buffer, ctrl->buffer_len);

            // Call real ioctl
            int result = real_ioctl(fd, request, arg);

            fprintf(stderr, "[TRACE] FW_CONTROL result: %d, response_len=%u\n",
                    result, ctrl->buffer_len);
            print_hex("Response buffer", ctrl->buffer, ctrl->buffer_len);

            return result;
        }
    }

    return real_ioctl(fd, request, arg);
}
```

Compile and use:

```bash
# Compile the interceptor
gcc -shared -fPIC -o /tmp/ioctl_trace.so /tmp/ioctl_trace.c -ldl

# Run Python test with interception
LD_PRELOAD=/tmp/ioctl_trace.so python3 /tmp/test_hailo.py 2>&1 | tee /tmp/hailo_trace.log

# Analyze the trace
grep -A20 "SET_NETWORK_GROUP_HEADER\|opcode.*32\|nr=0" /tmp/hailo_trace.log
```

### 3.2 Method B: Using Python to Trace Directly

Create `/tmp/trace_configure.py`:

```python
#!/usr/bin/env python3
"""
Traces the exact bytes sent during network group configuration.
"""
import sys
import ctypes
import struct

# Monkey-patch the ioctl to trace calls
original_ioctl = None

def traced_ioctl(fd, request, arg):
    global original_ioctl

    # Decode ioctl number
    nr = request & 0xff
    type_char = (request >> 8) & 0xff
    size = (request >> 16) & 0x3fff
    direction = (request >> 30) & 0x3

    print(f"\n=== IOCTL: fd={fd}, nr={nr}, type={chr(type_char)}, size={size}, dir={direction} ===")

    # If this looks like FW_CONTROL (nr=0, type='H')
    if type_char == ord('H') and nr == 0:
        # Read the hailo_fw_control structure
        # Offsets: md5=0, buffer_len=16, buffer=20, timeout=1520, cpu_id=1524
        class FwControl(ctypes.Structure):
            _fields_ = [
                ("expected_md5", ctypes.c_uint8 * 16),
                ("buffer_len", ctypes.c_uint32),
                ("buffer", ctypes.c_uint8 * 1500),
                ("timeout_ms", ctypes.c_uint32),
                ("cpu_id", ctypes.c_uint32),
            ]

        ctrl = ctypes.cast(arg, ctypes.POINTER(FwControl)).contents
        buf_len = ctrl.buffer_len
        cpu_id = ctrl.cpu_id

        print(f"FW_CONTROL: buffer_len={buf_len}, cpu_id={cpu_id}")
        print(f"Request hex: {bytes(ctrl.buffer[:min(buf_len, 80)]).hex()}")

        # Parse request header
        if buf_len >= 20:
            version, flags, seq, opcode = struct.unpack(">IIII", bytes(ctrl.buffer[:16]))
            param_count = struct.unpack(">I", bytes(ctrl.buffer[16:20]))[0]
            print(f"Header: version={version}, flags={flags}, seq={seq}, opcode={opcode}, params={param_count}")

            # For SET_NETWORK_GROUP_HEADER (opcode=32)
            if opcode == 32 and buf_len >= 56:
                app_header_len = struct.unpack(">I", bytes(ctrl.buffer[20:24]))[0]
                print(f"Application header length field: {app_header_len}")
                print(f"Application header data: {bytes(ctrl.buffer[24:56]).hex()}")

    # Call original ioctl
    result = original_ioctl(fd, request, arg)

    # Check response
    if type_char == ord('H') and nr == 0:
        ctrl = ctypes.cast(arg, ctypes.POINTER(FwControl)).contents
        print(f"Response: buffer_len={ctrl.buffer_len}")
        print(f"Response hex: {bytes(ctrl.buffer[:min(ctrl.buffer_len, 80)]).hex()}")

        if ctrl.buffer_len >= 24:
            version, flags, seq, opcode, major, minor = struct.unpack(">IIIIII", bytes(ctrl.buffer[:24]))
            print(f"Response: version={version}, seq={seq}, opcode={opcode}, major_status=0x{major:08x}, minor_status=0x{minor:08x}")

    return result

# Patch fcntl.ioctl
import fcntl
original_ioctl = fcntl.ioctl

def patched_ioctl(fd, request, arg=0, mutate_flag=True):
    if isinstance(arg, int):
        return original_ioctl(fd, request, arg, mutate_flag)

    # For buffer arguments, trace and call
    traced_ioctl(fd, request, ctypes.addressof(ctypes.c_char.from_buffer(arg)))
    return original_ioctl(fd, request, arg, mutate_flag)

fcntl.ioctl = patched_ioctl

# Now run the actual test
from hailo_platform import VDevice, HEF, ConfigureParams

hef_path = sys.argv[1] if len(sys.argv) > 1 else "/go-hailo/models/yolox_s_leaky_hailo8.hef"

print(f"Loading HEF: {hef_path}")

with VDevice() as vdevice:
    hef = HEF(hef_path)
    configure_params = ConfigureParams.create_from_hef(hef, interface=vdevice.get_default_interface())

    print("\n" + "="*60)
    print("CONFIGURING NETWORK GROUP - TRACE BELOW")
    print("="*60)

    network_group = vdevice.configure(hef, configure_params)[0]
    print(f"\nNetwork group configured: {network_group.name}")
```

### 3.3 Method C: Capture Raw Bytes with GDB

```bash
# Install gdb
sudo apt install gdb

# Create GDB script /tmp/trace_ioctl.gdb
cat > /tmp/trace_ioctl.gdb << 'EOF'
set pagination off
set logging file /tmp/gdb_trace.log
set logging on

# Break on ioctl
catch syscall ioctl

commands
  # Print ioctl arguments
  printf "ioctl: fd=%d, request=0x%lx\n", $rdi, $rsi

  # Check if it's Hailo FW_CONTROL (magic='H'=0x48, nr=0)
  if (($rsi >> 8) & 0xff) == 0x48
    printf "  Hailo ioctl, nr=%d\n", $rsi & 0xff
    # Print first 80 bytes of the buffer structure
    if ($rsi & 0xff) == 0
      printf "  FW_CONTROL buffer (first 80 bytes): "
      x/80xb $rdx+20
    end
  end
  continue
end

run
EOF

# Run with GDB
sudo gdb -x /tmp/trace_ioctl.gdb --args python3 /tmp/test_hailo.py
```

### 3.4 Compare SDK Output with Go Implementation

After capturing the SDK trace, compare with our Go output:

```bash
# Extract the SET_NETWORK_GROUP_HEADER request from SDK trace
grep -A5 "opcode=32" /tmp/hailo_trace.log > /tmp/sdk_header.txt

# Run Go implementation and capture its output
sudo ./bin/person-detector -model models/yolox_s_leaky_hailo8.hef test-images/three.jpg 2>&1 | grep -A5 "SetNetworkGroupHeader" > /tmp/go_header.txt

# Compare
diff /tmp/sdk_header.txt /tmp/go_header.txt
```

Key fields to compare:
1. **Request length** - Should be 56 bytes
2. **Application header length field** - SDK sends what value?
3. **Application header data** - All 32 bytes, byte-by-byte
4. **Byte order** - Verify length is big-endian

---

## Step 4: Check Hailo Community Forums

### 4.1 Search for Existing Issues

Search these resources for error code `0x40030060` or `INVALID_CONTEXT_SWITCH_APP_HEADER_LENGTH`:

1. **Hailo Community Forum**: https://community.hailo.ai/
   - Search: "INVALID_CONTEXT_SWITCH_APP_HEADER_LENGTH"
   - Search: "0x40030060"
   - Search: "set_network_group_header error"

2. **Hailo GitHub Issues**:
   - https://github.com/hailo-ai/hailort/issues
   - https://github.com/hailo-ai/hailort-drivers/issues

3. **Raspberry Pi Forums**:
   - https://forums.raspberrypi.com/ (search for "Hailo")

### 4.2 Post a Question (if needed)

If no existing answer is found, post with this information:

```
Title: SET_NETWORK_GROUP_HEADER returns INVALID_CONTEXT_SWITCH_APP_HEADER_LENGTH on Hailo8L

Environment:
- Device: Raspberry Pi 5 with Hailo8L (integrated NNC)
- Driver version: 4.20.0
- Device ID: 1e60:2864

Problem:
Sending SET_NETWORK_GROUP_HEADER (opcode 32) returns firmware error 0x40030060.
The application_header struct is packed to 32 bytes with #pragma pack(1).

Request format:
- 16-byte request header (version=2, flags=0, sequence, opcode=32)
- 4-byte param_count = 1 (big-endian)
- 4-byte application_header_length = 32 (big-endian)
- 32-byte application_header (little-endian struct)

Request hex:
00 00 00 02 00 00 00 00 00 00 00 02 00 00 00 20
00 00 00 01 00 00 00 20 02 00 00 00 00 00 01 00
02 01 00 00 00 00 00 00 00 00 00 00 00 00 00 00
00 00 00 00 00 00 00 00

Questions:
1. Is the struct layout correct for Hailo8L firmware?
2. Are there prerequisite commands needed before SET_NETWORK_GROUP_HEADER?
3. Does the integrated NNC use a different protocol than PCIe devices?
```

---

## Step 5: Alternative Approach - WRITE_ACTION_LIST ioctl

If the control protocol continues to fail, try using the driver's direct ioctl:

### 5.1 Understanding WRITE_ACTION_LIST

From `hailo_ioctl_common.h`:
```c
struct hailo_write_action_list_params {
    uint8_t *data;              // in: action list data
    size_t size;                // in: size of data
    uint64_t dma_address;       // out: DMA address
};
```

This ioctl writes action lists directly to device memory and returns a DMA address.

### 5.2 Implementation Approach

1. Parse action lists from HEF (they're in the protobuf data)
2. Write action lists using `WRITE_ACTION_LIST` ioctl
3. Set `external_action_list_address` in header to returned DMA address
4. Try `SET_NETWORK_GROUP_HEADER` with the DMA address set

### 5.3 Sample Implementation

```go
// In pkg/driver/ioctl.go

// WriteActionListParams matches struct hailo_write_action_list_params
type WriteActionListParams struct {
    Data       uintptr // pointer to data
    Size       uint64
    DmaAddress uint64  // output
}

func (d *DeviceFile) WriteActionList(data []byte) (uint64, error) {
    params := WriteActionListParams{
        Data: uintptr(unsafe.Pointer(&data[0])),
        Size: uint64(len(data)),
    }

    err := d.ioctl(ioctlWriteActionList, unsafe.Pointer(&params))
    if err != nil {
        return 0, err
    }

    return params.DmaAddress, nil
}
```

---

## Summary Checklist

- [ ] Install HailoRT SDK (`hailort`, `hailort-python3`)
- [ ] Verify device with `hailortcli fw-control identify`
- [ ] Run Python inference test to confirm device works
- [ ] Trace SDK's ioctl calls using LD_PRELOAD or GDB
- [ ] Capture exact bytes for SET_NETWORK_GROUP_HEADER from SDK
- [ ] Compare SDK bytes with Go implementation byte-by-byte
- [ ] Identify the difference (length field value, struct layout, etc.)
- [ ] Fix Go implementation to match SDK
- [ ] If stuck, check Hailo forums or post question
- [ ] Consider WRITE_ACTION_LIST as alternative approach

---

## Expected Outcome

After tracing the SDK, we expect to find one of:

1. **Different length value**: SDK might send a different value in the length field
2. **Different struct layout**: Firmware might expect additional fields or padding
3. **Different byte order**: Some fields might need different endianness
4. **Prerequisite state**: SDK might set up channels or other state first
5. **Different command sequence**: There might be other commands sent before the header

Once identified, update the Go implementation in `/go-hailo/pkg/control/protocol.go` accordingly.
