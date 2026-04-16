//go:build windows

package notify

import (
	"sync"
	"syscall"
	"unsafe"
)

var (
	user32dll   = syscall.NewLazyDLL("user32.dll")
	messageBoxW = user32dll.NewProc("MessageBoxW")

	// Allow only one popup at a time; drop new ones while one is visible.
	mu   sync.Mutex
	busy bool
)

const mbIconInformation = 0x00000040 // MB_ICONINFORMATION

// Show displays a native Windows MessageBox in a background goroutine.
// If a notification is already visible, the new one is silently dropped.
func Show(title, message string) {
	mu.Lock()
	if busy {
		mu.Unlock()
		return
	}
	busy = true
	mu.Unlock()

	go func() {
		defer func() {
			mu.Lock()
			busy = false
			mu.Unlock()
		}()

		titlePtr, _ := syscall.UTF16PtrFromString(title)
		msgPtr, _ := syscall.UTF16PtrFromString(message)
		messageBoxW.Call(
			0,
			uintptr(unsafe.Pointer(msgPtr)),
			uintptr(unsafe.Pointer(titlePtr)),
			mbIconInformation,
		)
	}()
}
