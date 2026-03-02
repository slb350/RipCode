package components

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestToast_New_InfoVariant(t *testing.T) {
	tm := NewToastManager()
	tm.Show("hello", ToastInfo, 3*time.Second)
	assert.NotNil(t, tm.Current())
	assert.Equal(t, ToastInfo, tm.Current().Variant)
}

func TestToast_View_ShowsMessage(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("Operation complete", ToastInfo, 3*time.Second)
	view := tm.View()
	assert.Contains(t, view, "Operation complete")
}

func TestToast_View_InfoStyle(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("info msg", ToastInfo, 3*time.Second)
	view := tm.View()
	assert.NotEmpty(t, view)
	assert.Contains(t, view, "info msg")
}

func TestToast_View_SuccessStyle(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("success msg", ToastSuccess, 3*time.Second)
	view := tm.View()
	assert.Contains(t, view, "success msg")
}

func TestToast_View_WarningStyle(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("warning msg", ToastWarning, 3*time.Second)
	view := tm.View()
	assert.Contains(t, view, "warning msg")
}

func TestToast_View_ErrorStyle(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	tm.Show("error msg", ToastError, 3*time.Second)
	view := tm.View()
	assert.Contains(t, view, "error msg")
}

func TestToast_Expired_ReturnsTrueAfterDuration(t *testing.T) {
	toast := Toast{
		Duration: 1 * time.Second,
		Created:  time.Now().Add(-2 * time.Second),
	}
	assert.True(t, toast.Expired())
}

func TestToast_Expired_ReturnsFalseBeforeDuration(t *testing.T) {
	toast := Toast{
		Duration: 5 * time.Second,
		Created:  time.Now(),
	}
	assert.False(t, toast.Expired())
}

func TestToastManager_Show_AddsToast(t *testing.T) {
	tm := NewToastManager()
	id := tm.Show("test", ToastInfo, 3*time.Second)
	assert.Greater(t, id, int64(0))
	assert.NotNil(t, tm.Current())
	assert.Equal(t, "test", tm.Current().Message)
}

func TestToastManager_Show_ReplacesExisting(t *testing.T) {
	tm := NewToastManager()
	id1 := tm.Show("first", ToastInfo, 3*time.Second)
	id2 := tm.Show("second", ToastWarning, 3*time.Second)
	assert.NotEqual(t, id1, id2)
	assert.Equal(t, "second", tm.Current().Message)
}

func TestToastManager_Dismiss_MatchingID(t *testing.T) {
	tm := NewToastManager()
	id := tm.Show("test", ToastInfo, 3*time.Second)
	tm.Dismiss(id)
	assert.Nil(t, tm.Current())
}

func TestToastManager_Dismiss_MismatchedID(t *testing.T) {
	tm := NewToastManager()
	tm.Show("first", ToastInfo, 3*time.Second)
	id2 := tm.Show("second", ToastWarning, 3*time.Second)
	// Dismiss with first ID (stale) should not dismiss second
	tm.Dismiss(id2 - 1)
	assert.NotNil(t, tm.Current(), "mismatched ID should not dismiss")
	assert.Equal(t, "second", tm.Current().Message)
}

func TestToastManager_View_Empty_ReturnsEmpty(t *testing.T) {
	tm := NewToastManager()
	tm.SetWidth(80)
	assert.Equal(t, "", tm.View())
}
