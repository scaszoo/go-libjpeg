// Package turbojpeg provides high-performance JPEG encoding and decoding
// using libjpeg-turbo's TurboJPEG 3.x API. Each operation crosses the
// cgo boundary at most 1-2 times, making it significantly faster than
// scanline-based bindings.
package turbojpeg

/*
#cgo pkg-config: libturbojpeg
#include "tj_helpers.h"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// PixelFormat controls the layout of interleaved pixel data.
type PixelFormat int

const (
	PixelFormatRGB  PixelFormat = C.TJPF_RGB
	PixelFormatRGBA PixelFormat = C.TJPF_RGBA
	PixelFormatGray PixelFormat = C.TJPF_GRAY
)

// BytesPerPixel returns the number of bytes per pixel for this format.
func (pf PixelFormat) BytesPerPixel() int {
	switch pf {
	case PixelFormatRGBA:
		return 4
	case PixelFormatRGB:
		return 3
	case PixelFormatGray:
		return 1
	default:
		return 0
	}
}

// Subsample controls chroma subsampling.
type Subsample int

const (
	Subsample444  Subsample = C.TJSAMP_444
	Subsample422  Subsample = C.TJSAMP_422
	Subsample420  Subsample = C.TJSAMP_420
	Subsample440  Subsample = C.TJSAMP_440
	SubsampleGray Subsample = C.TJSAMP_GRAY
)

// ScalingFactor for IDCT-based scaled decode.
type ScalingFactor struct {
	Num, Denom int
}

// ScalingFactors returns all scaling factors supported by the TurboJPEG library.
func ScalingFactors() []ScalingFactor {
	var n C.int
	ptr := C.tj3GetScalingFactors(&n)
	if ptr == nil || n == 0 {
		return nil
	}
	count := int(n)
	result := make([]ScalingFactor, count)
	// tjscalingfactor is {int num; int denom} = 8 bytes, walk the array manually
	base := unsafe.Pointer(ptr)
	sfSize := unsafe.Sizeof(C.tjscalingfactor{})
	for i := 0; i < count; i++ {
		sf := (*C.tjscalingfactor)(unsafe.Pointer(uintptr(base) + uintptr(i)*sfSize))
		result[i] = ScalingFactor{Num: int(sf.num), Denom: int(sf.denom)}
	}
	return result
}

// ImageHeader contains JPEG metadata without pixel data.
type ImageHeader struct {
	Width, Height int
	Subsample     Subsample
}

// DecodedImage is the result of decoding to an interleaved pixel format.
// Pix is a contiguous 1D byte slice of size Height * Stride.
type DecodedImage struct {
	Pix           []byte
	Width, Height int
	Stride        int
	PixelFormat   PixelFormat
}

// Rows returns a 2D view where each inner slice is one image row,
// sharing the same backing memory as Pix. No pixel data is copied.
func (img *DecodedImage) Rows() [][]byte {
	rows := make([][]byte, img.Height)
	bpp := img.PixelFormat.BytesPerPixel()
	for y := range rows {
		off := y * img.Stride
		rows[y] = img.Pix[off : off+img.Width*bpp]
	}
	return rows
}

// DecodedYCbCr is the result of decoding to separate Y/Cb/Cr planes.
type DecodedYCbCr struct {
	Y, Cb, Cr     []byte
	YStride       int
	CStride       int
	Width, Height int
	Subsample     Subsample
}

// YRows returns a 2D view into the Y plane.
func (img *DecodedYCbCr) YRows() [][]byte {
	return planeRows(img.Y, img.YStride, img.Height)
}

// CbRows returns a 2D view into the Cb plane.
func (img *DecodedYCbCr) CbRows() [][]byte {
	return planeRows(img.Cb, img.CStride, chromaHeight(img.Height, img.Subsample))
}

// CrRows returns a 2D view into the Cr plane.
func (img *DecodedYCbCr) CrRows() [][]byte {
	return planeRows(img.Cr, img.CStride, chromaHeight(img.Height, img.Subsample))
}

func planeRows(plane []byte, stride, height int) [][]byte {
	rows := make([][]byte, height)
	for y := range rows {
		off := y * stride
		rows[y] = plane[off : off+stride]
	}
	return rows
}

func chromaHeight(imgHeight int, subsamp Subsample) int {
	switch subsamp {
	case Subsample420, Subsample440:
		return (imgHeight + 1) / 2
	default:
		return imgHeight
	}
}

// QuantTable is a single 64-element quantization table in zig-zag order.
type QuantTable [64]uint16

// QuantTables holds the luminance and chrominance quantization tables.
type QuantTables struct {
	Luminance   QuantTable
	Chrominance QuantTable
}

// DecodeOptions controls decode behavior. All fields are optional.
type DecodeOptions struct {
	Scale  *ScalingFactor
	DstBuf []byte
}

// EncodeOptions controls encode behavior. All fields are optional.
type EncodeOptions struct {
	Quality     int
	Subsample   Subsample
	QuantTables *QuantTables
	DstBuf      []byte
}

func (o *EncodeOptions) quality() int {
	if o != nil && o.Quality > 0 {
		return o.Quality
	}
	return 85
}

func (o *EncodeOptions) subsample() Subsample {
	if o != nil && o.Subsample != 0 {
		return o.Subsample
	}
	return Subsample420
}

var errNilInput = errors.New("turbojpeg: nil or empty input")
