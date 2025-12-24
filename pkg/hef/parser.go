package hef

import (
	"fmt"
	"os"

	hefpb "github.com/anthropics/purple-hailo/pkg/hef/proto"
	"google.golang.org/protobuf/proto"
)

// Parse parses a HEF file from a file path
func Parse(path string) (*Hef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read HEF file: %w", err)
	}
	return ParseBytes(data)
}

// ParseBytes parses a HEF file from raw bytes
func ParseBytes(data []byte) (*Hef, error) {
	// Parse header
	header, err := ParseHeader(data)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header: %w", err)
	}

	// Get header size for this version
	headerSize, err := HeaderSize(header.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to get header size: %w", err)
	}

	// Extract protobuf region
	protoStart := headerSize
	protoEnd := protoStart + int(header.HefProtoSize)

	if protoEnd > len(data) {
		return nil, fmt.Errorf("%w: proto region exceeds file size", ErrTruncatedData)
	}

	protoData := data[protoStart:protoEnd]

	// Parse protobuf
	protoHef := &hefpb.ProtoHEFHef{}
	if err := proto.Unmarshal(protoData, protoHef); err != nil {
		return nil, fmt.Errorf("failed to parse protobuf: %w", err)
	}

	// Convert to our types
	hef := &Hef{
		Version:    header.Version,
		rawData:    data,
	}

	// Extract device architecture from proto header
	if protoHef.Header != nil {
		hef.DeviceArch = DeviceArchitecture(protoHef.Header.HwArch)
	}

	// Extract network groups
	for _, ng := range protoHef.NetworkGroups {
		ngInfo := extractNetworkGroupInfo(ng)
		hef.NetworkGroups = append(hef.NetworkGroups, ngInfo)
	}

	return hef, nil
}

// extractNetworkGroupInfo extracts information from a protobuf network group
func extractNetworkGroupInfo(ng *hefpb.ProtoHEFNetworkGroup) NetworkGroupInfo {
	info := NetworkGroupInfo{
		Name: ng.NetworkGroupName,
	}

	// Get metadata
	if ng.NetworkGroupMetadata != nil {
		info.BottleneckFps = ng.NetworkGroupMetadata.BottleneckFps
		// Use metadata name if main name is empty
		if info.Name == "" {
			info.Name = ng.NetworkGroupMetadata.NetworkGroupName
		}
	}

	// Extract streams from ops
	for _, op := range ng.Ops {
		if op.GetCoreOp() != nil {
			// This is a core op - extract streams from contexts
			coreOp := op.GetCoreOp()
			extractStreamsFromCoreOp(coreOp, &info)

			// Also extract configuration from core op
			if coreOp.PreliminaryConfig != nil {
				info.PreliminaryConfig = extractPreliminaryConfig(coreOp.PreliminaryConfig)
			}
			for _, ctx := range coreOp.Contexts {
				info.Contexts = append(info.Contexts, extractContextConfig(ctx))
			}
		}
	}

	// Also try to extract from contexts directly (for backward compatibility)
	for _, ctx := range ng.Contexts {
		if ctx.Metadata != nil {
			extractStreamsFromContext(ctx.Metadata, &info)
		}
	}

	// Extract preliminary config at network group level if present
	if ng.PreliminaryConfig != nil && info.PreliminaryConfig == nil {
		info.PreliminaryConfig = extractPreliminaryConfig(ng.PreliminaryConfig)
	}

	// Extract contexts at network group level if not already extracted
	if len(info.Contexts) == 0 {
		for _, ctx := range ng.Contexts {
			info.Contexts = append(info.Contexts, extractContextConfig(ctx))
		}
	}

	// Extract NMS op info if present
	for _, op := range ng.Ops {
		if nmsOp := op.GetNmsOp(); nmsOp != nil {
			// Found NMS operation - add to output vstreams
			vstream := extractNmsVStreamInfo(op.Name, nmsOp)
			info.OutputVStreams = append(info.OutputVStreams, vstream)
		}
	}

	return info
}

// extractPreliminaryConfig extracts preliminary configuration from protobuf
func extractPreliminaryConfig(pc *hefpb.ProtoHEFPreliminaryConfig) *PreliminaryConfig {
	if pc == nil {
		return nil
	}

	config := &PreliminaryConfig{}
	for _, op := range pc.Operation {
		config.Operations = append(config.Operations, extractConfigOperation(op))
	}
	return config
}

// extractContextConfig extracts context configuration from protobuf
func extractContextConfig(ctx *hefpb.ProtoHEFContext) ContextConfig {
	config := ContextConfig{
		Index: ctx.ContextIndex,
	}
	for _, op := range ctx.Operations {
		config.Operations = append(config.Operations, extractConfigOperation(op))
	}
	return config
}

// extractConfigOperation extracts a configuration operation from protobuf
func extractConfigOperation(op *hefpb.ProtoHEFOperation) ConfigOperation {
	config := ConfigOperation{}
	for _, action := range op.Actions {
		config.Actions = append(config.Actions, extractConfigAction(action))
	}
	return config
}

// extractConfigAction extracts a single configuration action from protobuf
func extractConfigAction(action *hefpb.ProtoHEFAction) ConfigAction {
	ca := ConfigAction{}

	switch a := action.Action.(type) {
	case *hefpb.ProtoHEFAction_WriteData:
		ca.Type = ActionTypeWriteData
		if a.WriteData != nil {
			ca.Address = a.WriteData.Address
			ca.Data = a.WriteData.Data
		}
	case *hefpb.ProtoHEFAction_WriteDataCcw:
		ca.Type = ActionTypeWriteDataCcw
		if a.WriteDataCcw != nil {
			// WriteDataCcw doesn't have an address, use channel index instead
			ca.Address = uint64(a.WriteDataCcw.CfgChannelIndex)
			ca.Data = a.WriteDataCcw.Data
		}
	case *hefpb.ProtoHEFAction_EnableSequencer:
		ca.Type = ActionTypeEnableSequencer
		if a.EnableSequencer != nil {
			initialL3Index := uint32(0)
			initialL3Offset := uint32(0)
			if a.EnableSequencer.InitialL3Info != nil {
				initialL3Index = a.EnableSequencer.InitialL3Info.InitialL3Index
				initialL3Offset = a.EnableSequencer.InitialL3Info.InitialL3Offset
			}
			ca.EnableSequencer = &EnableSequencerParams{
				ClusterIndex:    a.EnableSequencer.ClusterIndex,
				ActiveApuBitmap: a.EnableSequencer.ActiveApuBitmap,
				ActiveScBitmap:  a.EnableSequencer.ActiveScBitmap,
				ActiveL2Bitmap:  a.EnableSequencer.ActiveL2Bitmap,
				ActiveIaBitmap:  a.EnableSequencer.ActiveIaBitmap,
				L2Write0:        a.EnableSequencer.L2Write_0,
				L2Write1:        a.EnableSequencer.L2Write_1,
				L2Write2:        a.EnableSequencer.L2Write_2,
				L2Write3:        a.EnableSequencer.L2Write_3,
				InitialL3Index:  initialL3Index,
				InitialL3Offset: initialL3Offset,
			}
		}
	case *hefpb.ProtoHEFAction_WaitForSeqeuncer:
		ca.Type = ActionTypeWaitForSequencer
		if a.WaitForSeqeuncer != nil {
			// Store cluster index in Address field for now
			ca.Address = uint64(a.WaitForSeqeuncer.ClusterIndex)
		}
	case *hefpb.ProtoHEFAction_DisableLcu:
		ca.Type = ActionTypeDisableLcu
		if a.DisableLcu != nil {
			ca.DisableLcu = &DisableLcuParams{
				LcuIndex:         a.DisableLcu.LcuIndex,
				ClusterIndex:     a.DisableLcu.ClusterIndex,
				LcuEnableAddress: a.DisableLcu.LcuEnableAddress,
			}
		}
	case *hefpb.ProtoHEFAction_EnableLcu:
		ca.Type = ActionTypeEnableLcu
		if a.EnableLcu != nil {
			ca.EnableLcu = &EnableLcuParams{
				LcuIndex:          a.EnableLcu.LcuIndex,
				ClusterIndex:      a.EnableLcu.ClusterIndex,
				KernelDoneAddress: a.EnableLcu.LcuKernelDoneAddress,
				KernelDoneCount:   a.EnableLcu.LcuKernelDoneCount,
				LcuEnableAddress:  a.EnableLcu.LcuEnableAddress,
				NetworkIndex:      a.EnableLcu.NetworkIndex,
			}
		}
	case *hefpb.ProtoHEFAction_None:
		ca.Type = ActionTypeNone
	case *hefpb.ProtoHEFAction_AllowInputDataflow:
		ca.Type = ActionTypeAllowInputDataflow
		if a.AllowInputDataflow != nil {
			ca.Address = uint64(a.AllowInputDataflow.SysIndex)
		}
	case *hefpb.ProtoHEFAction_WaitForModuleConfigDone:
		ca.Type = ActionTypeWaitForModuleConfigDone
		if a.WaitForModuleConfigDone != nil {
			ca.Address = uint64(a.WaitForModuleConfigDone.Index)
		}
	case *hefpb.ProtoHEFAction_EnableNms:
		ca.Type = ActionTypeEnableNms
		if a.EnableNms != nil {
			ca.EnableNms = &EnableNmsParams{
				NmsUnitIndex:    a.EnableNms.NmsUnitIndex,
				NetworkIndex:    a.EnableNms.NetworkIndex,
				NumberOfClasses: a.EnableNms.NumberOfClasses,
				BurstSize:       a.EnableNms.BurstSize,
				DivisionFactor:  a.EnableNms.DivisionFactor,
			}
		}
	case *hefpb.ProtoHEFAction_WriteDataByType:
		ca.Type = ActionTypeWriteDataByType
		if a.WriteDataByType != nil {
			ca.Address = a.WriteDataByType.Address
			ca.Data = a.WriteDataByType.Data
			ca.WriteDataByType = &WriteDataByTypeParams{
				Address:      a.WriteDataByType.Address,
				DataType:     uint32(a.WriteDataByType.DataType),
				Data:         a.WriteDataByType.Data,
				Mask:         a.WriteDataByType.Mask,
				NetworkIndex: a.WriteDataByType.NetworkIndex,
				Shift:        a.WriteDataByType.Shift,
			}
		}
	case *hefpb.ProtoHEFAction_SwitchLcuBatch:
		ca.Type = ActionTypeSwitchLcuBatch
		if a.SwitchLcuBatch != nil {
			ca.SwitchLcuBatch = &SwitchLcuBatchParams{
				LcuIndex:     a.SwitchLcuBatch.LcuIndex,
				ClusterIndex: a.SwitchLcuBatch.ClusterIndex,
				NetworkIndex: a.SwitchLcuBatch.NetworkIndex,
			}
		}
	default:
		ca.Type = ActionTypeNone
	}

	return ca
}

// extractStreamsFromCoreOp extracts streams from a core operation
func extractStreamsFromCoreOp(coreOp *hefpb.ProtoHEFCoreOp, info *NetworkGroupInfo) {
	for _, ctx := range coreOp.Contexts {
		if ctx.Metadata != nil {
			extractStreamsFromContext(ctx.Metadata, info)
		}
	}
}

// extractStreamsFromContext extracts streams from a context
func extractStreamsFromContext(meta *hefpb.ProtoHEFContextMetadata, info *NetworkGroupInfo) {
	for _, edgeLayer := range meta.EdgeLayers {
		layerInfo := edgeLayer.GetLayerInfo()
		if layerInfo == nil {
			continue
		}

		stream := extractStreamInfo(layerInfo, edgeLayer.Direction)

		if edgeLayer.Direction == hefpb.ProtoHEFEdgeLayerDirection_PROTO__EDGE_LAYER_DIRECTION__HOST_TO_DEVICE {
			// Check for duplicates
			found := false
			for _, existing := range info.InputStreams {
				if existing.Name == stream.Name {
					found = true
					break
				}
			}
			if !found {
				info.InputStreams = append(info.InputStreams, stream)
			}
		} else {
			// Check for duplicates
			found := false
			for _, existing := range info.OutputStreams {
				if existing.Name == stream.Name {
					found = true
					break
				}
			}
			if !found {
				info.OutputStreams = append(info.OutputStreams, stream)
			}
		}
	}
}

// extractStreamInfo extracts stream info from an edge layer
func extractStreamInfo(layer *hefpb.ProtoHEFEdgeLayerInfo, dir hefpb.ProtoHEFEdgeLayerDirection) StreamInfo {
	stream := StreamInfo{
		Name: layer.Name,
	}

	if dir == hefpb.ProtoHEFEdgeLayerDirection_PROTO__EDGE_LAYER_DIRECTION__HOST_TO_DEVICE {
		stream.Direction = StreamDirectionInput
	} else {
		stream.Direction = StreamDirectionOutput
	}

	base := layer.EdgeLayerBase
	if base != nil {
		stream.Shape = ImageShape3D{
			Height:   base.Height,
			Width:    base.Width,
			Features: base.Features,
		}
		stream.HwShape = ImageShape3D{
			Height:   base.PaddedHeight,
			Width:    base.PaddedWidth,
			Features: base.PaddedFeatures,
		}
		stream.HwFrameSize = uint64(base.CoreBytesPerBuffer) * uint64(base.CoreBuffersPerFrame)

		// Extract format
		stream.Format = extractFormat(base)
	}

	// Extract quantization info
	if layer.NumericInfo != nil {
		stream.QuantInfo = QuantInfo{
			ZeroPoint: layer.NumericInfo.QpZp,
			Scale:     layer.NumericInfo.QpScale,
			LimMin:    layer.NumericInfo.LimvalsMin,
			LimMax:    layer.NumericInfo.LimvalsMax,
		}
	}

	return stream
}

// extractFormat extracts format information from edge layer base
func extractFormat(base *hefpb.ProtoHEFEdgeLayerBase) Format {
	format := Format{}

	// Data type
	switch base.DataBytes {
	case 1:
		format.Type = FormatTypeUint8
	case 2:
		format.Type = FormatTypeUint16
	default:
		format.Type = FormatTypeUint8
	}

	// Format order
	switch base.Format {
	case uint32(hefpb.ProtoHEFFormatOrder_PROTO__FORMAT__ORDER__NHWC):
		format.Order = FormatOrderNHWC
	case uint32(hefpb.ProtoHEFFormatOrder_PROTO__FORMAT__ORDER__NCHW):
		format.Order = FormatOrderNCHW
	case uint32(hefpb.ProtoHEFFormatOrder_PROTO__FORMAT__ORDER__NV12):
		format.Order = FormatOrderNV12
	default:
		format.Order = FormatOrderNHWC
	}

	return format
}

// extractNmsVStreamInfo extracts NMS vstream info from an NMS op
func extractNmsVStreamInfo(name string, nmsOp *hefpb.ProtoHEFNmsOp) VStreamInfo {
	vstream := VStreamInfo{
		Name:      name,
		Direction: StreamDirectionOutput,
		IsNms:     true,
		Format: Format{
			Order: FormatOrderHailoNmsByClass,
		},
		NmsShape: NmsShape{
			NumberOfClasses:   nmsOp.Classes,
			MaxBboxesPerClass: nmsOp.MaxProposalsPerClass,
		},
	}

	// Extract NMS type-specific info
	if yoloxOp := nmsOp.GetYoloxNmsOp(); yoloxOp != nil {
		// YOLOX NMS - used in person detector
		vstream.Shape = ImageShape3D{
			Height: uint32(yoloxOp.ImageHeight),
			Width:  uint32(yoloxOp.ImageWidth),
		}
	} else if yoloOp := nmsOp.GetYoloNmsOp(); yoloOp != nil {
		// YOLOv5 NMS
		vstream.Shape = ImageShape3D{
			Height: uint32(yoloOp.ImageHeight),
			Width:  uint32(yoloOp.ImageWidth),
		}
	} else if yolov8Op := nmsOp.GetYolov8NmsOp(); yolov8Op != nil {
		// YOLOv8 NMS
		vstream.Shape = ImageShape3D{
			Height: uint32(yolov8Op.ImageHeight),
			Width:  uint32(yolov8Op.ImageWidth),
		}
	}

	return vstream
}

// GetNetworkGroup returns a network group by name
func (h *Hef) GetNetworkGroup(name string) (*NetworkGroupInfo, error) {
	for i := range h.NetworkGroups {
		if h.NetworkGroups[i].Name == name {
			return &h.NetworkGroups[i], nil
		}
	}
	return nil, fmt.Errorf("network group %q not found", name)
}

// GetDefaultNetworkGroup returns the first network group
func (h *Hef) GetDefaultNetworkGroup() (*NetworkGroupInfo, error) {
	if len(h.NetworkGroups) == 0 {
		return nil, fmt.Errorf("no network groups in HEF")
	}
	return &h.NetworkGroups[0], nil
}

// GetInputStreamInfo returns input stream info by name
func (ng *NetworkGroupInfo) GetInputStreamInfo(name string) (*StreamInfo, error) {
	for i := range ng.InputStreams {
		if ng.InputStreams[i].Name == name {
			return &ng.InputStreams[i], nil
		}
	}
	return nil, fmt.Errorf("input stream %q not found", name)
}

// GetUserInputs returns only user-facing input streams (excludes internal buffers)
func (ng *NetworkGroupInfo) GetUserInputs() []StreamInfo {
	var inputs []StreamInfo
	for _, stream := range ng.InputStreams {
		// Skip internal context buffers
		if isInternalStream(stream.Name) {
			continue
		}
		inputs = append(inputs, stream)
	}
	return inputs
}

// GetUserOutputs returns only user-facing output streams (excludes intermediate)
func (ng *NetworkGroupInfo) GetUserOutputs() []StreamInfo {
	var outputs []StreamInfo
	for _, stream := range ng.OutputStreams {
		// Skip internal context buffers
		if isInternalStream(stream.Name) {
			continue
		}
		outputs = append(outputs, stream)
	}
	return outputs
}

// isInternalStream checks if a stream name indicates an internal buffer
func isInternalStream(name string) bool {
	// Internal streams typically have "context_X_to_context_Y" in their name
	if len(name) > 0 {
		for i := 0; i < len(name)-7; i++ {
			if name[i:i+7] == "context" {
				return true
			}
		}
	}
	return false
}

// GetOutputStreamInfo returns output stream info by name
func (ng *NetworkGroupInfo) GetOutputStreamInfo(name string) (*StreamInfo, error) {
	for i := range ng.OutputStreams {
		if ng.OutputStreams[i].Name == name {
			return &ng.OutputStreams[i], nil
		}
	}
	return nil, fmt.Errorf("output stream %q not found", name)
}

// InputFrameSize returns the total input frame size in bytes
func (ng *NetworkGroupInfo) InputFrameSize() int {
	total := 0
	for _, stream := range ng.InputStreams {
		size := int(stream.Shape.Height * stream.Shape.Width * stream.Shape.Features)
		// Adjust for data type
		switch stream.Format.Type {
		case FormatTypeUint16:
			size *= 2
		case FormatTypeFloat32:
			size *= 4
		}
		total += size
	}
	return total
}

// OutputFrameSize returns the total output frame size in bytes
func (ng *NetworkGroupInfo) OutputFrameSize() int {
	total := 0
	for _, stream := range ng.OutputStreams {
		size := int(stream.Shape.Height * stream.Shape.Width * stream.Shape.Features)
		// Adjust for data type
		switch stream.Format.Type {
		case FormatTypeUint16:
			size *= 2
		case FormatTypeFloat32:
			size *= 4
		}
		total += size
	}
	return total
}

// HasNmsOutput returns true if any output is an NMS layer
func (ng *NetworkGroupInfo) HasNmsOutput() bool {
	for _, vs := range ng.OutputVStreams {
		if vs.IsNms {
			return true
		}
	}
	return false
}

// GetNmsInfo returns NMS configuration if present
func (ng *NetworkGroupInfo) GetNmsInfo() *VStreamInfo {
	for i := range ng.OutputVStreams {
		if ng.OutputVStreams[i].IsNms {
			return &ng.OutputVStreams[i]
		}
	}
	return nil
}
