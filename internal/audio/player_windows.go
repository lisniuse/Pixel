//go:build windows

// Package audio plays audio data through the Windows winmm API.
package audio

import (
	"runtime"
	"sync"
	"syscall"
	"unsafe"
)

var (
	winmm     = syscall.NewLazyDLL("winmm.dll")
	playSound = winmm.NewProc("PlaySoundW")

	// Only one clip plays at a time; new requests are dropped while busy.
	mu   sync.Mutex
	busy bool
)

const (
	sndSync      = 0x0000 // play synchronously (block until done)
	sndNoDefault = 0x0002 // don't play default sound on error
	sndMemory    = 0x0004 // pszSound points to an in-memory RIFF/WAV blob
)

// Play plays a WAV audio blob asynchronously in a background goroutine.
// If playback is already in progress the call is silently dropped.
// The WAV data must be a complete RIFF/WAV file (as returned by the TTS API
// when encoding="wav").
func Play(wavData []byte) {
	if len(wavData) == 0 {
		return
	}

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

		// PlaySoundW with SND_MEMORY | SND_SYNC:
		//   - first arg is a pointer to the in-memory WAV blob
		//   - blocks until playback is finished, keeping wavData alive
		playSound.Call(
			uintptr(unsafe.Pointer(&wavData[0])),
			0,
			uintptr(sndSync|sndMemory|sndNoDefault),
		)
		// Prevent the GC from collecting wavData before PlaySoundW returns.
		runtime.KeepAlive(wavData)
	}()
}
