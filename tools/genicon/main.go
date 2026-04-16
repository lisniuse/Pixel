// genicon renders assets/icon.svg into a multi-resolution ICO file.
//
// Usage (run from project root):
//
//	go run ./tools/genicon/ [-svg assets/icon.svg] [-out internal/tray/assets/icon.ico]
package main

import (
	"bytes"
	"encoding/binary"
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func main() {
	svgPath := flag.String("svg", "assets/icon.svg", "input SVG file")
	outPath := flag.String("out", "internal/tray/assets/icon.ico", "output ICO file")
	flag.Parse()

	// --- 1. Parse SVG ---
	f, err := os.Open(*svgPath)
	if err != nil {
		fatalf("open %s: %v", *svgPath, err)
	}
	defer f.Close()

	icon, err := oksvg.ReadIconStream(f)
	if err != nil {
		fatalf("parse svg: %v", err)
	}

	// --- 2. Render PNG for each ICO size ---
	sizes := []int{256, 48, 32, 16}
	pngs := make([][]byte, len(sizes))

	for i, size := range sizes {
		icon.SetTarget(0, 0, float64(size), float64(size))
		img := image.NewRGBA(image.Rect(0, 0, size, size))
		scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
		dasher := rasterx.NewDasher(size, size, scanner)
		icon.Draw(dasher, 1.0)

		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			fatalf("encode png@%d: %v", size, err)
		}
		pngs[i] = buf.Bytes()
		fmt.Printf("  rendered %dx%d  (%d bytes PNG)\n", size, size, buf.Len())
	}

	// --- 3. Pack into ICO ---
	icoData := buildICO(sizes, pngs)

	if err := os.MkdirAll(filepath.Dir(*outPath), 0o755); err != nil {
		fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(*outPath, icoData, 0o644); err != nil {
		fatalf("write %s: %v", *outPath, err)
	}
	fmt.Printf("Generated %s (%d bytes)\n", *outPath, len(icoData))
}

// buildICO packs one PNG per size into a Windows ICO container.
// Windows Vista+ allows raw PNG payloads inside ICO files, so no BMP
// conversion is needed.
func buildICO(sizes []int, pngs [][]byte) []byte {
	n := len(sizes)
	var buf bytes.Buffer

	// ICONDIR header (6 bytes)
	writeU16(&buf, 0)          // reserved
	writeU16(&buf, 1)          // type = 1 (ICO)
	writeU16(&buf, uint16(n))  // image count

	// First image data starts right after all ICONDIRENTRY records.
	offset := uint32(6 + n*16)

	// ICONDIRENTRY × n (16 bytes each)
	for i, size := range sizes {
		w, h := uint8(size), uint8(size)
		if size >= 256 {
			w, h = 0, 0 // ICO encodes 256 as 0
		}
		writeU8(&buf, w)
		writeU8(&buf, h)
		writeU8(&buf, 0)           // color count (0 = truecolor)
		writeU8(&buf, 0)           // reserved
		writeU16(&buf, 1)          // color planes
		writeU16(&buf, 32)         // bits per pixel
		writeU32(&buf, uint32(len(pngs[i])))
		writeU32(&buf, offset)
		offset += uint32(len(pngs[i]))
	}

	// Image payloads
	for _, p := range pngs {
		buf.Write(p)
	}

	return buf.Bytes()
}

func writeU8(b *bytes.Buffer, v uint8)  { b.WriteByte(v) }
func writeU16(b *bytes.Buffer, v uint16) { _ = binary.Write(b, binary.LittleEndian, v) }
func writeU32(b *bytes.Buffer, v uint32) { _ = binary.Write(b, binary.LittleEndian, v) }

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "genicon: "+format+"\n", args...)
	os.Exit(1)
}
