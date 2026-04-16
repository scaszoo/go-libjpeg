package turbojpeg

/*
#include "quant.c"
*/
import "C"
import (
	"errors"
	"unsafe"
)

// ExtractQuantTables reads quantization tables from JPEG data without full decode.
func ExtractQuantTables(jpegData []byte) (*QuantTables, error) {
	if len(jpegData) == 0 {
		return nil, errNilInput
	}

	var lumTable, chromTable [64]C.ushort
	var hasChrom C.int
	var errBuf [C.ERR_BUF_SIZE]C.char

	ret := C.extract_quant_tables(
		(*C.uchar)(unsafe.Pointer(&jpegData[0])),
		C.size_t(len(jpegData)),
		&lumTable[0], &chromTable[0],
		&hasChrom,
		&errBuf[0], C.int(C.ERR_BUF_SIZE),
	)
	if ret != 0 {
		return nil, errors.New("turbojpeg: " + C.GoString(&errBuf[0]))
	}

	qt := &QuantTables{}
	for i := 0; i < 64; i++ {
		qt.Luminance[i] = uint16(lumTable[i])
	}
	if hasChrom != 0 {
		for i := 0; i < 64; i++ {
			qt.Chrominance[i] = uint16(chromTable[i])
		}
	}
	return qt, nil
}
