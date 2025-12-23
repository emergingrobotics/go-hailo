# HailoRT Driver IOCTL Reference

This document describes the official Hailo driver IOCTL interface based on analysis of the
`hailort-drivers` repository (version 4.23.0).

## Overview

The Hailo driver uses several IOCTL magic numbers for different subsystems:

| Magic | Character | Subsystem |
|-------|-----------|-----------|
| `HAILO_GENERAL_IOCTL_MAGIC` | `'g'` | General device queries |
| `HAILO_VDMA_IOCTL_MAGIC` | `'v'` | VDMA operations |
| `HAILO_NNC_IOCTL_MAGIC` | `'n'` | Neural Network Core operations |
| `HAILO_SOC_IOCTL_MAGIC` | `'s'` | SoC operations |
| `HAILO_PCI_EP_IOCTL_MAGIC` | `'p'` | PCIe endpoint operations |

All structures are packed with `#pragma pack(push, 1)`, meaning NO padding between fields.

## Critical Constants

```c
#define MAX_VDMA_CHANNELS_PER_ENGINE            (32)
#define VDMA_CHANNELS_PER_ENGINE_PER_DIRECTION  (16)
#define MAX_VDMA_ENGINES                        (3)
#define SIZE_OF_VDMA_DESCRIPTOR                 (16)
#define VDMA_DEST_CHANNELS_START                (16)
#define HAILO_VDMA_MAX_ONGOING_TRANSFERS        (128)
#define CHANNEL_IRQ_TIMESTAMPS_SIZE             (256)  // HAILO_VDMA_MAX_ONGOING_TRANSFERS * 2
#define HAILO_MAX_BUFFERS_PER_SINGLE_TRANSFER   (8)
#define MAX_CONTROL_LENGTH                      (1500)
#define MAX_NOTIFICATION_LENGTH                 (1500)
#define PCIE_EXPECTED_MD5_LENGTH                (16)
```

---

## VDMA IOCTL Structures

### hailo_vdma_interrupts_channel_data (3 bytes)

```c
struct hailo_vdma_interrupts_channel_data {
    uint8_t engine_index;    // offset 0, 1 byte
    uint8_t channel_index;   // offset 1, 1 byte
    uint8_t data;            // offset 2, 1 byte
};
// Total: 3 bytes
```

The `data` field contains either:
- Number of transfers completed (0-253)
- `HAILO_VDMA_TRANSFER_DATA_CHANNEL_NOT_ACTIVE` (0xff) - channel not active
- `HAILO_VDMA_TRANSFER_DATA_CHANNEL_WITH_ERROR` (0xfe) - channel error

### hailo_vdma_interrupts_wait_params (301 bytes)

```c
struct hailo_vdma_interrupts_wait_params {
    uint32_t channels_bitmap_per_engine[MAX_VDMA_ENGINES];          // offset 0,  12 bytes (in)
    uint8_t channels_count;                                         // offset 12, 1 byte   (out)
    struct hailo_vdma_interrupts_channel_data
        irq_data[MAX_VDMA_CHANNELS_PER_ENGINE * MAX_VDMA_ENGINES];  // offset 13, 288 bytes (out)
};
// Total: 12 + 1 + (32 * 3 * 3) = 301 bytes
```

**IOCTL**: `HAILO_VDMA_INTERRUPTS_WAIT = _IOWR_('v', 2, struct hailo_vdma_interrupts_wait_params)`

### hailo_vdma_transfer_buffer (16 bytes)

```c
struct hailo_vdma_transfer_buffer {
    enum hailo_dma_buffer_type buffer_type; // offset 0, 4 bytes (HAILO_DMA_USER_PTR_BUFFER=0, HAILO_DMA_DMABUF_BUFFER=1)
    uintptr_t addr_or_fd;                   // offset 4, 8 bytes (user address or dmabuf fd)
    uint32_t size;                          // offset 12, 4 bytes
};
// Total: 16 bytes
```

### hailo_vdma_launch_transfer_params (153 bytes)

```c
struct hailo_vdma_launch_transfer_params {
    uint8_t engine_index;                                               // offset 0,   1 byte
    uint8_t channel_index;                                              // offset 1,   1 byte
    uintptr_t desc_handle;                                              // offset 2,   8 bytes
    uint32_t starting_desc;                                             // offset 10,  4 bytes
    bool should_bind;                                                   // offset 14,  1 byte
    uint8_t buffers_count;                                              // offset 15,  1 byte
    struct hailo_vdma_transfer_buffer buffers[8];                       // offset 16,  128 bytes (8 * 16)
    enum hailo_vdma_interrupts_domain first_interrupts_domain;          // offset 144, 4 bytes
    enum hailo_vdma_interrupts_domain last_interrupts_domain;           // offset 148, 4 bytes
    bool is_debug;                                                      // offset 152, 1 byte
};
// Total: 153 bytes (NO OUTPUT FIELDS!)
```

**IOCTL**: `HAILO_VDMA_LAUNCH_TRANSFER = _IOR_('v', 15, struct hailo_vdma_launch_transfer_params)`

**Important**: This is `_IOR_` (read from user), NOT `_IOWR_`. The kernel reads the params but does NOT write back any output.

### hailo_desc_list_program_params (51 bytes)

```c
struct hailo_desc_list_program_params {
    size_t buffer_handle;                                              // offset 0,  8 bytes
    size_t buffer_size;                                                // offset 8,  8 bytes
    size_t buffer_offset;                                              // offset 16, 8 bytes
    uint32_t batch_size;                                               // offset 24, 4 bytes  <-- IMPORTANT!
    uintptr_t desc_handle;                                             // offset 28, 8 bytes
    uint8_t channel_index;                                             // offset 36, 1 byte
    uint32_t starting_desc;                                            // offset 37, 4 bytes
    bool should_bind;                                                  // offset 41, 1 byte
    enum hailo_vdma_interrupts_domain last_interrupts_domain;          // offset 42, 4 bytes
    bool is_debug;                                                     // offset 46, 1 byte
    uint32_t stride;                                                   // offset 47, 4 bytes  <-- IMPORTANT!
};
// Total: 51 bytes
```

**IOCTL**: `HAILO_DESC_LIST_PROGRAM = _IOR_('v', 9, struct hailo_desc_list_program_params)`

### hailo_desc_list_create_params (27 bytes)

```c
struct hailo_desc_list_create_params {
    size_t desc_count;          // offset 0,  8 bytes (in)
    uint16_t desc_page_size;    // offset 8,  2 bytes (in)
    bool is_circular;           // offset 10, 1 byte  (in)
    uintptr_t desc_handle;      // offset 11, 8 bytes (out)
    uint64_t dma_address;       // offset 19, 8 bytes (out)
};
// Total: 27 bytes
```

**IOCTL**: `HAILO_DESC_LIST_CREATE = _IOWR_('v', 7, struct hailo_desc_list_create_params)`

### hailo_vdma_buffer_map_params (40 bytes)

```c
struct hailo_vdma_buffer_map_params {
    uintptr_t user_address;                         // offset 0,  8 bytes (in)
    size_t size;                                    // offset 8,  8 bytes (in)
    enum hailo_dma_data_direction data_direction;   // offset 16, 4 bytes (in)
    enum hailo_dma_buffer_type buffer_type;         // offset 20, 4 bytes (in)
    uintptr_t allocated_buffer_handle;              // offset 24, 8 bytes (in)
    size_t mapped_handle;                           // offset 32, 8 bytes (out)
};
// Total: 40 bytes
```

**IOCTL**: `HAILO_VDMA_BUFFER_MAP = _IOWR_('v', 4, struct hailo_vdma_buffer_map_params)`

### hailo_vdma_enable_channels_params (13 bytes)

```c
struct hailo_vdma_enable_channels_params {
    uint32_t channels_bitmap_per_engine[MAX_VDMA_ENGINES];  // offset 0,  12 bytes (in)
    bool enable_timestamps_measure;                         // offset 12, 1 byte   (in)
};
// Total: 13 bytes
```

**IOCTL**: `HAILO_VDMA_ENABLE_CHANNELS = _IOR_('v', 0, struct hailo_vdma_enable_channels_params)`

### hailo_vdma_disable_channels_params (12 bytes)

```c
struct hailo_vdma_disable_channels_params {
    uint32_t channels_bitmap_per_engine[MAX_VDMA_ENGINES];  // offset 0, 12 bytes (in)
};
// Total: 12 bytes
```

**IOCTL**: `HAILO_VDMA_DISABLE_CHANNELS = _IOR_('v', 1, struct hailo_vdma_disable_channels_params)`

---

## NNC IOCTL Structures

### hailo_write_action_list_params (24 bytes)

```c
struct hailo_write_action_list_params {
    uint8_t *data;              // offset 0,  8 bytes (in) - pointer to data
    size_t size;                // offset 8,  8 bytes (in)
    uint64_t dma_address;       // offset 16, 8 bytes (out)
};
// Total: 24 bytes
```

**IOCTL**: `HAILO_WRITE_ACTION_LIST = _IOW_('n', 5, struct hailo_write_action_list_params)`

**Note**: This is `_IOW_` (write to user), NOT `_IOWR_`.

---

## Interrupts Domain Enum

```c
enum hailo_vdma_interrupts_domain {
    HAILO_VDMA_INTERRUPTS_DOMAIN_NONE   = 0,
    HAILO_VDMA_INTERRUPTS_DOMAIN_DEVICE = (1 << 0),  // 1
    HAILO_VDMA_INTERRUPTS_DOMAIN_HOST   = (1 << 1),  // 2
    // BOTH = DEVICE | HOST = 3
};
```

---

## Inference Flow

The official HailoRT library performs inference as follows:

### 1. Device Initialization
```
1. Open /dev/hailo0
2. HAILO_QUERY_DRIVER_INFO - validate driver version
3. HAILO_QUERY_DEVICE_PROPERTIES - get device capabilities
```

### 2. Buffer Setup
```
1. Allocate user-space buffers for input/output
2. HAILO_VDMA_BUFFER_MAP - map buffers for DMA access
3. HAILO_DESC_LIST_CREATE - create descriptor lists for each channel
```

### 3. Network Configuration
```
1. Parse HEF file to extract network configuration
2. HAILO_FW_CONTROL - send configuration to firmware
3. HAILO_WRITE_ACTION_LIST - write action lists for each context
```

### 4. Channel Setup
```
1. HAILO_DESC_LIST_PROGRAM - program descriptor lists with buffer info
   - Must include batch_size and stride for proper operation
2. HAILO_VDMA_ENABLE_CHANNELS - enable the required channels
```

### 5. Inference Loop
```
For each frame:
1. Copy input data to mapped buffer
2. HAILO_VDMA_BUFFER_SYNC (SYNC_FOR_DEVICE) - sync input buffer
3. HAILO_VDMA_LAUNCH_TRANSFER - launch H2D transfer (input)
4. HAILO_VDMA_LAUNCH_TRANSFER - launch D2H transfer (output)
5. HAILO_VDMA_INTERRUPTS_WAIT - wait for completion
6. HAILO_VDMA_BUFFER_SYNC (SYNC_FOR_CPU) - sync output buffer
7. Read output data from mapped buffer
```

### 6. Cleanup
```
1. HAILO_VDMA_DISABLE_CHANNELS - disable channels
2. HAILO_DESC_LIST_RELEASE - release descriptor lists
3. HAILO_VDMA_BUFFER_UNMAP - unmap buffers
4. Close device
```

---

## Key Differences from Naive Implementation

1. **Packed Structures**: All structures use `#pragma pack(1)` - no padding!
2. **Buffer Count**: `HAILO_MAX_BUFFERS_PER_SINGLE_TRANSFER = 8`, not 2
3. **Transfer Buffer Layout**: Uses `buffer_type + addr_or_fd + size`, NOT `handle + offset + size`
4. **DescListProgram**: Requires `batch_size` and `stride` fields
5. **LaunchTransfer**: Is `_IOR_` (no output), not `_IOWR_`
6. **InterruptsWait irq_data**: 3 bytes per entry, not 7
