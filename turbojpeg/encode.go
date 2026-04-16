package turbojpeg

/*
#include "encode.c"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// EncodeRGBA encodes RGBA pixels to JPEG.
func EncodeRGBA(pix []byte, width, height, stride int, opts *EncodeOptions) ([]byte, error) {
	return encodeInterleaved(pix, width, height, stride, PixelFormatRGBA, opts)
}

// EncodeRGB encodes RGB pixels to JPEG.
func EncodeRGB(pix []byte, width, height, stride int, opts *EncodeOptions) ([]byte, error) {
	return encodeInterleaved(pix, width, height, stride, PixelFormatRGB, opts)
}

// EncodeGray encodes grayscale pixels to JPEG.
// Subsample is automatically set to SubsampleGray if not specified.
func EncodeGray(pix []byte, width, height, stride int, opts *EncodeOptions) ([]byte, error) {
	if opts == nil {
		opts = &EncodeOptions{Subsample: SubsampleGray}
	} else if opts.Subsample == 0 {
		cp := *opts
		cp.Subsample = SubsampleGray
		opts = &cp
	}
	return encodeInterleaved(pix, width, height, stride, PixelFormatGray, opts)
}

func encodeInterleaved(pix []byte, width, height, stride int, pf PixelFormat, opts *EncodeOptions) ([]byte, error) {
	if len(pix) == 0 {
		return nil, errNilInput
	}

	subsamp := opts.subsample()
	var errBuf [C.ERR_BUF_SIZE]C.char

	// If custom quant tables are provided, use the jpeglib-based path
	if opts != nil && opts.QuantTables != nil {
		var lumTable, chromTable [64]C.ushort
		for i := 0; i < 64; i++ {
			lumTable[i] = C.ushort(opts.QuantTables.Luminance[i])
			chromTable[i] = C.ushort(opts.QuantTables.Chrominance[i])
		}

		var outBuf *C.uchar
		var outSize C.ulong

		ret := C.encode_interleaved_with_quant(
			(*C.uchar)(unsafe.Pointer(&pix[0])),
			C.int(width), C.int(stride), C.int(height),
			C.int(pf), C.int(subsamp),
			&lumTable[0], &chromTable[0],
			&outBuf, &outSize,
			&errBuf[0], C.int(C.ERR_BUF_SIZE),
		)
		if ret != 0 {
			return nil, errors.New("turbojpeg: " + C.GoString(&errBuf[0]))
		}
		defer C.free(unsafe.Pointer(outBuf))
		return copyResult(outBuf, outSize, opts), nil
	}

	// Standard TurboJPEG path (quality-based)
	quality := opts.quality()
	var outBuf *C.uchar
	var outSize C.ulong

	ret := C.encode_interleaved(
		(*C.uchar)(unsafe.Pointer(&pix[0])),
		C.int(width), C.int(stride), C.int(height),
		C.int(pf),
		C.int(quality), C.int(subsamp),
		&outBuf, &outSize,
		&errBuf[0], C.int(C.ERR_BUF_SIZE),
	)
	if ret != 0 {
		return nil, errors.New("turbojpeg: " + C.GoString(&errBuf[0]))
	}
	defer C.tj3Free(unsafe.Pointer(outBuf))
	return copyResult(outBuf, outSize, opts), nil
}

// copyResult copies C output into Go slice, reusing DstBuf if available.
func copyResult(outBuf *C.uchar, outSize C.ulong, opts *EncodeOptions) []byte {
	size := int(outSize)
	if opts != nil && len(opts.DstBuf) >= size {
		C.memcpy(unsafe.Pointer(&opts.DstBuf[0]), unsafe.Pointer(outBuf), C.size_t(size))
		return opts.DstBuf[:size]
	}
	return C.GoBytes(unsafe.Pointer(outBuf), C.int(outSize))
}

// EncodeYCbCr encodes Y/Cb/Cr planes to JPEG.
func EncodeYCbCr(y, cb, cr []byte, yStride, cStride int,
	width, height int, subsample Subsample, opts *EncodeOptions) ([]byte, error) {
	if len(y) == 0 {
		return nil, errNilInput
	}

	quality := opts.quality()

	var cbPtr, crPtr *C.uchar
	if len(cb) > 0 {
		cbPtr = (*C.uchar)(unsafe.Pointer(&cb[0]))
		crPtr = (*C.uchar)(unsafe.Pointer(&cr[0]))
	}

	var outBuf *C.uchar
	var outSize C.ulong
	var errBuf [C.ERR_BUF_SIZE]C.char

	ret := C.encode_yuv_planes(
		(*C.uchar)(unsafe.Pointer(&y[0])),
		cbPtr, crPtr,
		C.int(yStride), C.int(cStride),
		C.int(width), C.int(height), C.int(subsample),
		C.int(quality),
		&outBuf, &outSize,
		&errBuf[0], C.int(C.ERR_BUF_SIZE),
	)
	if ret != 0 {
		return nil, errors.New("turbojpeg: " + C.GoString(&errBuf[0]))
	}
	defer C.tj3Free(unsafe.Pointer(outBuf))
	return copyResult(outBuf, outSize, opts), nil
}
