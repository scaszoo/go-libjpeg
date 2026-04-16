package turbojpeg

/*
#include "reencode.c"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// Reencode decodes a JPEG and re-encodes it with the given options.
// The decode-to-RGB and encode-from-RGB happens entirely in C-land.
// This is the most efficient path for quality assessment pipelines.
// If opts.QuantTables is set, custom quantization tables are used.
func Reencode(jpegData []byte, opts *EncodeOptions) ([]byte, error) {
	if len(jpegData) == 0 {
		return nil, errNilInput
	}

	quality := opts.quality()
	subsamp := opts.subsample()

	var lumPtr, chromPtr *C.ushort
	var lumTable, chromTable [64]C.ushort
	if opts != nil && opts.QuantTables != nil {
		for i := 0; i < 64; i++ {
			lumTable[i] = C.ushort(opts.QuantTables.Luminance[i])
			chromTable[i] = C.ushort(opts.QuantTables.Chrominance[i])
		}
		lumPtr = &lumTable[0]
		chromPtr = &chromTable[0]
	}

	var outBuf *C.uchar
	var outSize C.ulong
	var errBuf [C.ERR_BUF_SIZE]C.char

	ret := C.reencode(
		(*C.uchar)(unsafe.Pointer(&jpegData[0])),
		C.size_t(len(jpegData)),
		&outBuf, &outSize,
		C.int(quality), C.int(subsamp),
		lumPtr, chromPtr,
		&errBuf[0], C.int(C.ERR_BUF_SIZE),
	)
	if ret != 0 {
		return nil, errors.New("turbojpeg: " + C.GoString(&errBuf[0]))
	}

	// Free depends on path: quant tables use malloc (jpeg_mem_dest),
	// standard uses tj3Alloc
	if opts != nil && opts.QuantTables != nil {
		defer C.free(unsafe.Pointer(outBuf))
	} else {
		defer C.tj3Free(unsafe.Pointer(outBuf))
	}

	return C.GoBytes(unsafe.Pointer(outBuf), C.int(outSize)), nil
}
