package turbojpeg

import (
	"os"
	"testing"
)

func loadTestJPEG(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("testdata/cosmos.jpg")
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func TestDecodeHeader(t *testing.T) {
	data := loadTestJPEG(t)
	hdr, err := DecodeHeader(data)
	if err != nil {
		t.Fatal(err)
	}
	if hdr.Width == 0 || hdr.Height == 0 {
		t.Fatalf("unexpected zero dimensions: %dx%d", hdr.Width, hdr.Height)
	}
	t.Logf("Header: %dx%d subsample=%d", hdr.Width, hdr.Height, hdr.Subsample)
}

func TestDecodeHeader_NilInput(t *testing.T) {
	_, err := DecodeHeader(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestDecodeRGBA(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeRGBA(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if img.Width == 0 || img.Height == 0 {
		t.Fatalf("unexpected zero dimensions: %dx%d", img.Width, img.Height)
	}
	expectedSize := img.Height * img.Stride
	if len(img.Pix) != expectedSize {
		t.Fatalf("Pix size %d != expected %d", len(img.Pix), expectedSize)
	}
	if img.PixelFormat != PixelFormatRGBA {
		t.Fatalf("expected RGBA pixel format, got %d", img.PixelFormat)
	}
	t.Logf("DecodeRGBA: %dx%d stride=%d pixbytes=%d", img.Width, img.Height, img.Stride, len(img.Pix))
}

func TestDecodeRGB(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeRGB(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if img.Stride != img.Width*3 {
		t.Fatalf("RGB stride %d != width*3 %d", img.Stride, img.Width*3)
	}
}

func TestDecodeGray(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeGray(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if img.Stride != img.Width {
		t.Fatalf("Gray stride %d != width %d", img.Stride, img.Width)
	}
}

func TestDecodeRGBA_Scaled(t *testing.T) {
	data := loadTestJPEG(t)
	hdr, err := DecodeHeader(data)
	if err != nil {
		t.Fatal(err)
	}

	scale := &ScalingFactor{Num: 1, Denom: 2}
	img, err := DecodeRGBA(data, &DecodeOptions{Scale: scale})
	if err != nil {
		t.Fatal(err)
	}

	expectW := (hdr.Width*scale.Num + scale.Denom - 1) / scale.Denom
	expectH := (hdr.Height*scale.Num + scale.Denom - 1) / scale.Denom
	if img.Width != expectW || img.Height != expectH {
		t.Fatalf("scaled dims %dx%d != expected %dx%d", img.Width, img.Height, expectW, expectH)
	}
}

func TestDecodeRGBA_BufferReuse(t *testing.T) {
	data := loadTestJPEG(t)
	img1, err := DecodeRGBA(data, nil)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, len(img1.Pix))
	img2, err := DecodeRGBA(data, &DecodeOptions{DstBuf: buf})
	if err != nil {
		t.Fatal(err)
	}

	if img1.Width != img2.Width || img1.Height != img2.Height {
		t.Fatalf("dimensions mismatch: %dx%d vs %dx%d", img1.Width, img1.Height, img2.Width, img2.Height)
	}
	for i := range img1.Pix {
		if img1.Pix[i] != img2.Pix[i] {
			t.Fatalf("pixel mismatch at offset %d: %d vs %d", i, img1.Pix[i], img2.Pix[i])
		}
	}
}

func TestDecodeRGBA_Rows(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeRGBA(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	rows := img.Rows()
	if len(rows) != img.Height {
		t.Fatalf("Rows() returned %d rows, expected %d", len(rows), img.Height)
	}
	if len(rows[0]) != img.Width*4 {
		t.Fatalf("row length %d != width*4 %d", len(rows[0]), img.Width*4)
	}
	if &rows[0][0] != &img.Pix[0] {
		t.Fatal("Rows() does not share backing memory with Pix")
	}
}

func TestDecodeYCbCr(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeYCbCr(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	if img.Width == 0 || img.Height == 0 {
		t.Fatalf("unexpected zero dimensions: %dx%d", img.Width, img.Height)
	}
	if len(img.Y) == 0 {
		t.Fatal("Y plane is empty")
	}
	t.Logf("DecodeYCbCr: %dx%d yStride=%d cStride=%d subsamp=%d Y=%d Cb=%d Cr=%d",
		img.Width, img.Height, img.YStride, img.CStride, img.Subsample,
		len(img.Y), len(img.Cb), len(img.Cr))
}

func TestDecodeYCbCr_YRows(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeYCbCr(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	yRows := img.YRows()
	if len(yRows) != img.Height {
		t.Fatalf("YRows() returned %d rows, expected %d", len(yRows), img.Height)
	}
	if &yRows[0][0] != &img.Y[0] {
		t.Fatal("YRows() does not share backing memory with Y")
	}
}

func TestDecodeCorruptData(t *testing.T) {
	_, err := DecodeRGBA([]byte{0xFF, 0xD8, 0x00, 0x00}, nil)
	if err == nil {
		t.Fatal("expected error for corrupt data")
	}
}
