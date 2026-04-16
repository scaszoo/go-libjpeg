package turbojpeg

/*
#include "decode.c"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// DecodeHeader reads JPEG dimensions and subsampling without decoding pixels.
func DecodeHeader(jpegData []byte) (*ImageHeader, error) {
	if len(jpegData) == 0 {
		return nil, errNilInput
	}
	var w, h, subsamp C.int
	var errBuf [C.ERR_BUF_SIZE]C.char

	ret := C.decode_header(
		(*C.uchar)(unsafe.Pointer(&jpegData[0])),
		C.size_t(len(jpegData)),
		&w, &h, &subsamp,
		&errBuf[0], C.int(C.ERR_BUF_SIZE),
	)
	if ret != 0 {
		return nil, errors.New("turbojpeg: " + C.GoString(&errBuf[0]))
	}
	return &ImageHeader{
		Width:     int(w),
		Height:    int(h),
		Subsample: Subsample(subsamp),
	}, nil
}

// DecodeRGBA decodes JPEG to interleaved RGBA pixels (4 bytes/pixel).
func DecodeRGBA(jpegData []byte, opts *DecodeOptions) (*DecodedImage, error) {
	return decodeInterleaved(jpegData, PixelFormatRGBA, opts)
}

// DecodeRGB decodes JPEG to interleaved RGB pixels (3 bytes/pixel).
func DecodeRGB(jpegData []byte, opts *DecodeOptions) (*DecodedImage, error) {
	return decodeInterleaved(jpegData, PixelFormatRGB, opts)
}

// DecodeGray decodes JPEG to grayscale pixels (1 byte/pixel).
func DecodeGray(jpegData []byte, opts *DecodeOptions) (*DecodedImage, error) {
	return decodeInterleaved(jpegData, PixelFormatGray, opts)
}

func decodeInterleaved(jpegData []byte, pf PixelFormat, opts *DecodeOptions) (*DecodedImage, error) {
	if len(jpegData) == 0 {
		return nil, errNilInput
	}

	scaleNum, scaleDenom := 0, 0
	if opts != nil && opts.Scale != nil {
		scaleNum = opts.Scale.Num
		scaleDenom = opts.Scale.Denom
	}

	jpegPtr := (*C.uchar)(unsafe.Pointer(&jpegData[0]))
	jpegSize := C.size_t(len(jpegData))

	var dstBuf []byte
	if opts != nil && len(opts.DstBuf) > 0 {
		dstBuf = opts.DstBuf
	} else {
		var w, h, stride C.int
		var errBuf [C.ERR_BUF_SIZE]C.char
		ret := C.decode_header_dims(
			jpegPtr, jpegSize,
			C.int(pf),
			C.int(scaleNum), C.int(scaleDenom),
			&w, &h, &stride,
			&errBuf[0], C.int(C.ERR_BUF_SIZE),
		)
		if ret != 0 {
			return nil, errors.New("turbojpeg: " + C.GoString(&errBuf[0]))
		}
		dstBuf = make([]byte, int(h)*int(stride))
	}

	var outW, outH, outStride C.int
	var errBuf [C.ERR_BUF_SIZE]C.char

	ret := C.decode_interleaved(
		jpegPtr, jpegSize,
		(*C.uchar)(unsafe.Pointer(&dstBuf[0])),
		&outW, &outH, &outStride,
		C.int(pf),
		C.int(scaleNum), C.int(scaleDenom),
		&errBuf[0], C.int(C.ERR_BUF_SIZE),
	)
	if ret != 0 {
		return nil, errors.New("turbojpeg: " + C.GoString(&errBuf[0]))
	}

	needed := int(outH) * int(outStride)
	return &DecodedImage{
		Pix:         dstBuf[:needed],
		Width:       int(outW),
		Height:      int(outH),
		Stride:      int(outStride),
		PixelFormat: pf,
	}, nil
}
