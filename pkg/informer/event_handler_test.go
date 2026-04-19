package informer

import (
	"testing"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

// mockEventHandler tracks which methods were called
type mockEventHandler struct {
	addCalls    []corev1.Pod
	updateCalls [][]corev1.Pod // pairs of [old, new]
	deleteCalls []corev1.Pod
}

func (m *mockEventHandler) OnAdd(obj corev1.Pod) {
	m.addCalls = append(m.addCalls, obj)
}

func (m *mockEventHandler) OnUpdate(oldObj, newObj corev1.Pod) {
	m.updateCalls = append(m.updateCalls, []corev1.Pod{oldObj, newObj})
}

func (m *mockEventHandler) OnDelete(obj corev1.Pod) {
	m.deleteCalls = append(m.deleteCalls, obj)
}

func createTestPod(name, namespace string) *unstructured.Unstructured {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "test", Image: "test:latest"}},
		},
	}

	unstructuredObj, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(pod)
	return &unstructured.Unstructured{Object: unstructuredObj}
}

func TestInformerEventHandlers(t *testing.T) {
	// Create informer with mock handler
	inf := &informer[corev1.Pod]{}
	handler := &mockEventHandler{}

	err := inf.AddEventHandler(handler)
	if err != nil {
		t.Fatalf("Failed to add event handler: %v", err)
	}

	testPod := createTestPod("test-pod", "test-ns")

	// Test OnAdd
	inf.OnAdd(testPod, false)
	if len(handler.addCalls) != 1 {
		t.Errorf("Expected 1 OnAdd call, got %d", len(handler.addCalls))
	}
	if handler.addCalls[0].Name != "test-pod" {
		t.Errorf("Expected pod name 'test-pod', got '%s'", handler.addCalls[0].Name)
	}

	// Test OnUpdate
	updatedPod := createTestPod("test-pod", "test-ns")
	inf.OnUpdate(testPod, updatedPod)
	if len(handler.updateCalls) != 1 {
		t.Errorf("Expected 1 OnUpdate call, got %d", len(handler.updateCalls))
	}
	if len(handler.updateCalls[0]) != 2 {
		t.Errorf("Expected OnUpdate call with 2 pods, got %d", len(handler.updateCalls[0]))
	}

	// Test OnDelete - this is the critical test for the bug fix
	inf.OnDelete(testPod)
	if len(handler.deleteCalls) != 1 {
		t.Errorf("Expected 1 OnDelete call, got %d", len(handler.deleteCalls))
	}
	if handler.deleteCalls[0].Name != "test-pod" {
		t.Errorf("Expected deleted pod name 'test-pod', got '%s'", handler.deleteCalls[0].Name)
	}

	// Verify OnDelete didn't accidentally call OnAdd (the original bug)
	if len(handler.addCalls) != 1 {
		t.Errorf("OnDelete incorrectly called OnAdd - expected 1 add call total, got %d", len(handler.addCalls))
	}
}

func TestAddEventHandlerNil(t *testing.T) {
	inf := &informer[corev1.Pod]{}
	err := inf.AddEventHandler(nil)
	if err == nil {
		t.Error("Expected error when adding nil handler")
	}
}

func TestInformerEventHandlersWithInvalidData(t *testing.T) {
	// Test that informer handles invalid unstructured data gracefully
	inf := &informer[corev1.Pod]{}
	handler := &mockEventHandler{}
	inf.AddEventHandler(handler)

	// Create invalid unstructured object that can't be converted to Pod
	invalidObj := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"spec": map[string]interface{}{
				"containers": "this-should-be-array-not-string", // Invalid: containers should be array
			},
			"metadata": map[string]interface{}{
				"name":      "test",
				"namespace": "test",
			},
		},
	}

	// These should not panic and should not call handler methods due to conversion errors
	inf.OnAdd(invalidObj, false)
	inf.OnUpdate(invalidObj, invalidObj)
	inf.OnDelete(invalidObj)

	// Verify no handler methods were called due to conversion failures
	if len(handler.addCalls) != 0 {
		t.Errorf("Expected 0 OnAdd calls with invalid data, got %d", len(handler.addCalls))
	}
	if len(handler.updateCalls) != 0 {
		t.Errorf("Expected 0 OnUpdate calls with invalid data, got %d", len(handler.updateCalls))
	}
	if len(handler.deleteCalls) != 0 {
		t.Errorf("Expected 0 OnDelete calls with invalid data, got %d", len(handler.deleteCalls))
	}
}