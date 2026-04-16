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

func TestEncodeRGBA(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeRGBA(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	jpeg, err := EncodeRGBA(img.Pix, img.Width, img.Height, img.Stride,
		&EncodeOptions{Quality: 90})
	if err != nil {
		t.Fatal(err)
	}
	if len(jpeg) == 0 {
		t.Fatal("encoded JPEG is empty")
	}
	if jpeg[0] != 0xFF || jpeg[1] != 0xD8 {
		t.Fatalf("invalid JPEG header: %02x %02x", jpeg[0], jpeg[1])
	}
	t.Logf("EncodeRGBA: %d bytes (input %d pixels)", len(jpeg), len(img.Pix))
}

func TestEncodeRGB(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeRGB(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	jpeg, err := EncodeRGB(img.Pix, img.Width, img.Height, img.Stride, nil)
	if err != nil {
		t.Fatal(err)
	}
	if jpeg[0] != 0xFF || jpeg[1] != 0xD8 {
		t.Fatalf("invalid JPEG header: %02x %02x", jpeg[0], jpeg[1])
	}
}

func TestEncodeGray(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeGray(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	jpeg, err := EncodeGray(img.Pix, img.Width, img.Height, img.Stride, nil)
	if err != nil {
		t.Fatal(err)
	}
	if jpeg[0] != 0xFF || jpeg[1] != 0xD8 {
		t.Fatalf("invalid JPEG header: %02x %02x", jpeg[0], jpeg[1])
	}
}

func TestEncodeYCbCr(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeYCbCr(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	jpeg, err := EncodeYCbCr(img.Y, img.Cb, img.Cr, img.YStride, img.CStride,
		img.Width, img.Height, img.Subsample, &EncodeOptions{Quality: 85})
	if err != nil {
		t.Fatal(err)
	}
	if jpeg[0] != 0xFF || jpeg[1] != 0xD8 {
		t.Fatalf("invalid JPEG header: %02x %02x", jpeg[0], jpeg[1])
	}
	t.Logf("EncodeYCbCr: %d bytes", len(jpeg))
}

func TestEncodeRGBA_DstBufReuse(t *testing.T) {
	data := loadTestJPEG(t)
	img, err := DecodeRGBA(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	jpeg1, err := EncodeRGBA(img.Pix, img.Width, img.Height, img.Stride,
		&EncodeOptions{Quality: 85})
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]byte, len(jpeg1)*2)
	jpeg2, err := EncodeRGBA(img.Pix, img.Width, img.Height, img.Stride,
		&EncodeOptions{Quality: 85, DstBuf: buf})
	if err != nil {
		t.Fatal(err)
	}
	if len(jpeg1) != len(jpeg2) {
		t.Fatalf("size mismatch: %d vs %d", len(jpeg1), len(jpeg2))
	}
	if &jpeg2[0] != &buf[0] {
		t.Fatal("DstBuf was not reused")
	}
}

func TestRoundtrip_RGBA(t *testing.T) {
	data := loadTestJPEG(t)
	img1, err := DecodeRGBA(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	jpeg, err := EncodeRGBA(img1.Pix, img1.Width, img1.Height, img1.Stride,
		&EncodeOptions{Quality: 100})
	if err != nil {
		t.Fatal(err)
	}
	img2, err := DecodeRGBA(jpeg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if img1.Width != img2.Width || img1.Height != img2.Height {
		t.Fatalf("dimensions changed: %dx%d -> %dx%d",
			img1.Width, img1.Height, img2.Width, img2.Height)
	}
	maxDiff := 0
	for i := range img1.Pix {
		d := int(img1.Pix[i]) - int(img2.Pix[i])
		if d < 0 {
			d = -d
		}
		if d > maxDiff {
			maxDiff = d
		}
	}
	if maxDiff > 30 {
		t.Fatalf("round-trip max pixel diff %d exceeds tolerance 30", maxDiff)
	}
	t.Logf("round-trip max pixel diff: %d", maxDiff)
}

func TestExtractQuantTables(t *testing.T) {
	data := loadTestJPEG(t)
	qt, err := ExtractQuantTables(data)
	if err != nil {
		t.Fatal(err)
	}
	allZero := true
	for _, v := range qt.Luminance {
		if v != 0 {
			allZero = false
			break
		}
	}
	if allZero {
		t.Fatal("luminance table is all zeros")
	}
	t.Logf("Luminance[0]=%d Luminance[63]=%d", qt.Luminance[0], qt.Luminance[63])
	t.Logf("Chrominance[0]=%d Chrominance[63]=%d", qt.Chrominance[0], qt.Chrominance[63])
}

func TestExtractQuantTables_NilInput(t *testing.T) {
	_, err := ExtractQuantTables(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}

func TestEncodeRGBA_WithQuantTables(t *testing.T) {
	data := loadTestJPEG(t)
	qt, err := ExtractQuantTables(data)
	if err != nil {
		t.Fatal(err)
	}
	img, err := DecodeRGBA(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	jpeg, err := EncodeRGBA(img.Pix, img.Width, img.Height, img.Stride,
		&EncodeOptions{QuantTables: qt})
	if err != nil {
		t.Fatal(err)
	}
	if jpeg[0] != 0xFF || jpeg[1] != 0xD8 {
		t.Fatalf("invalid JPEG header: %02x %02x", jpeg[0], jpeg[1])
	}
	qt2, err := ExtractQuantTables(jpeg)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 64; i++ {
		if qt.Luminance[i] != qt2.Luminance[i] {
			t.Fatalf("luminance table mismatch at [%d]: %d vs %d",
				i, qt.Luminance[i], qt2.Luminance[i])
		}
	}
	t.Logf("QuantTable round-trip verified: tables match")
}

func TestReencode(t *testing.T) {
	data := loadTestJPEG(t)
	out, err := Reencode(data, &EncodeOptions{Quality: 75})
	if err != nil {
		t.Fatal(err)
	}
	if len(out) == 0 {
		t.Fatal("reencode output is empty")
	}
	if out[0] != 0xFF || out[1] != 0xD8 {
		t.Fatalf("invalid JPEG header: %02x %02x", out[0], out[1])
	}
	t.Logf("Reencode: input=%d bytes, output=%d bytes (Q75)", len(data), len(out))
}

func TestReencode_MatchesManualRoundtrip(t *testing.T) {
	data := loadTestJPEG(t)
	opts := &EncodeOptions{Quality: 80, Subsample: Subsample420}

	reencoded, err := Reencode(data, opts)
	if err != nil {
		t.Fatal(err)
	}

	img, err := DecodeRGB(data, nil)
	if err != nil {
		t.Fatal(err)
	}
	manual, err := EncodeRGB(img.Pix, img.Width, img.Height, img.Stride, opts)
	if err != nil {
		t.Fatal(err)
	}

	if len(reencoded) != len(manual) {
		t.Fatalf("size mismatch: reencode=%d manual=%d", len(reencoded), len(manual))
	}
	for i := range reencoded {
		if reencoded[i] != manual[i] {
			t.Fatalf("byte mismatch at offset %d", i)
		}
	}
}

func TestReencode_NilInput(t *testing.T) {
	_, err := Reencode(nil, nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
}
