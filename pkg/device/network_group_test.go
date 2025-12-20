//go:build unit

package device

import (
	"sync"
	"testing"

	"github.com/anthropics/purple-hailo/pkg/hef"
)

// createMockConfiguredNetworkGroup creates a mock configured network group for testing
func createMockConfiguredNetworkGroup(name string, isMultiContext bool) *ConfiguredNetworkGroup {
	return &ConfiguredNetworkGroup{
		info: &hef.NetworkGroupInfo{
			Name:           name,
			IsMultiContext: isMultiContext,
		},
		state: StateConfigured,
		inputs: []StreamInfo{
			{Name: "input0", FrameSize: 224 * 224 * 3, Height: 224, Width: 224, Channels: 3},
		},
		outputs: []StreamInfo{
			{Name: "output0", FrameSize: 1000, Height: 1, Width: 1, Channels: 1000},
		},
	}
}

func TestNetworkGroupCreate(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("yolo_network", false)

	if ng.Name() != "yolo_network" {
		t.Errorf("Name() = %s, expected yolo_network", ng.Name())
	}

	if ng.State() != StateConfigured {
		t.Errorf("State() = %d, expected StateConfigured", ng.State())
	}
}

func TestNetworkGroupStreamInfos(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test_network", false)

	inputs := ng.InputStreamInfos()
	outputs := ng.OutputStreamInfos()

	if len(inputs) != 1 {
		t.Errorf("expected 1 input stream, got %d", len(inputs))
	}

	if len(outputs) != 1 {
		t.Errorf("expected 1 output stream, got %d", len(outputs))
	}

	if inputs[0].Name != "input0" {
		t.Errorf("input name = %s, expected input0", inputs[0].Name)
	}

	if inputs[0].FrameSize != 224*224*3 {
		t.Errorf("input frame size = %d, expected %d", inputs[0].FrameSize, 224*224*3)
	}
}

func TestNetworkGroupMultiContext(t *testing.T) {
	singleContext := createMockConfiguredNetworkGroup("single", false)
	multiContext := createMockConfiguredNetworkGroup("multi", true)

	if singleContext.IsMultiContext() {
		t.Error("single context network should not be multi-context")
	}

	if !multiContext.IsMultiContext() {
		t.Error("multi context network should be multi-context")
	}
}

func TestNetworkGroupActivate(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	ang, err := ng.Activate()
	if err != nil {
		t.Fatalf("Activate() error = %v", err)
	}

	if ng.State() != StateActivated {
		t.Errorf("State() = %d, expected StateActivated", ng.State())
	}

	if !ang.IsActive() {
		t.Error("activated network group should be active")
	}
}

func TestNetworkGroupDeactivate(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	ang, _ := ng.Activate()

	err := ang.Deactivate()
	if err != nil {
		t.Fatalf("Deactivate() error = %v", err)
	}

	if ng.State() != StateDeactivated {
		t.Errorf("State() = %d, expected StateDeactivated", ng.State())
	}

	if ang.IsActive() {
		t.Error("deactivated network group should not be active")
	}
}

func TestNetworkGroupDoubleActivate(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	_, err := ng.Activate()
	if err != nil {
		t.Fatalf("first Activate() error = %v", err)
	}

	_, err = ng.Activate()
	if err != ErrInvalidState {
		t.Errorf("second Activate() should return ErrInvalidState, got %v", err)
	}
}

func TestNetworkGroupReactivate(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	// Activate
	ang, _ := ng.Activate()

	// Deactivate
	ang.Deactivate()

	// Should be able to activate again
	ang2, err := ng.Activate()
	if err != nil {
		t.Fatalf("re-activate failed: %v", err)
	}

	if !ang2.IsActive() {
		t.Error("reactivated network should be active")
	}
}

func TestNetworkGroupDoubleDeactivate(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	ang, _ := ng.Activate()

	err := ang.Deactivate()
	if err != nil {
		t.Errorf("first Deactivate() error = %v", err)
	}

	err = ang.Deactivate()
	if err != nil {
		t.Errorf("second Deactivate() should be safe: %v", err)
	}
}

func TestNetworkGroupCloseWhileActivated(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	ng.Activate()

	err := ng.Close()
	if err != ErrStillActivated {
		t.Errorf("Close() while activated should return ErrStillActivated, got %v", err)
	}
}

func TestNetworkGroupCloseAfterDeactivate(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	ang, _ := ng.Activate()
	ang.Deactivate()

	err := ng.Close()
	if err != nil {
		t.Errorf("Close() after deactivate should succeed: %v", err)
	}
}

func TestNetworkGroupConcurrentActivate(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	var wg sync.WaitGroup
	var successCount int
	var mu sync.Mutex

	numGoroutines := 10
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			_, err := ng.Activate()
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	// Only one should succeed
	if successCount != 1 {
		t.Errorf("expected exactly 1 successful activation, got %d", successCount)
	}
}

func TestNetworkGroupStateTransitions(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	// Initial state: Configured
	if ng.State() != StateConfigured {
		t.Errorf("initial state = %d, expected Configured", ng.State())
	}

	// Activate -> Activated
	ang, _ := ng.Activate()
	if ng.State() != StateActivated {
		t.Errorf("after activate: state = %d, expected Activated", ng.State())
	}

	// Deactivate -> Deactivated
	ang.Deactivate()
	if ng.State() != StateDeactivated {
		t.Errorf("after deactivate: state = %d, expected Deactivated", ng.State())
	}

	// Re-activate -> Activated
	ang2, _ := ng.Activate()
	if ng.State() != StateActivated {
		t.Errorf("after re-activate: state = %d, expected Activated", ng.State())
	}

	// Deactivate and close -> Uninitialized
	ang2.Deactivate()
	ng.Close()
	if ng.State() != StateUninitialized {
		t.Errorf("after close: state = %d, expected Uninitialized", ng.State())
	}
}

func TestNetworkGroupInputShape(t *testing.T) {
	ng := createMockConfiguredNetworkGroup("test", false)

	inputs := ng.InputStreamInfos()
	if len(inputs) == 0 {
		t.Fatal("no input streams")
	}

	input := inputs[0]
	expectedSize := uint64(input.Height * input.Width * input.Channels)

	if input.FrameSize != expectedSize {
		t.Errorf("FrameSize = %d, expected %d (H*W*C)", input.FrameSize, expectedSize)
	}
}

// Benchmarks

func BenchmarkNetworkGroupActivateDeactivate(b *testing.B) {
	ng := createMockConfiguredNetworkGroup("bench", false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ang, _ := ng.Activate()
		ang.Deactivate()
	}
}

func BenchmarkNetworkGroupStreamInfoAccess(b *testing.B) {
	ng := createMockConfiguredNetworkGroup("bench", false)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ng.InputStreamInfos()
		_ = ng.OutputStreamInfos()
	}
}
