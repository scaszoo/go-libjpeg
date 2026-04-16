#include "tj_helpers.h"
#include <jpeglib.h>

static int encode_interleaved(
    const unsigned char *src_buf, int width, int pitch, int height,
    int pixel_format,
    int quality, int subsamp,
    unsigned char **out_buf, unsigned long *out_size,
    char *err_buf, int err_buf_size)
{
    tjhandle h = tj3Init(TJINIT_COMPRESS);
    if (!h) {
        snprintf(err_buf, err_buf_size, "tj3Init failed");
        return -1;
    }

    if (tj3Set(h, TJPARAM_QUALITY, quality) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }
    if (tj3Set(h, TJPARAM_SUBSAMP, subsamp) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    *out_buf = NULL;
    size_t jpegSize = 0;

    if (tj3Compress8(h, src_buf, width, pitch, height, pixel_format,
                     out_buf, &jpegSize) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    *out_size = (unsigned long)jpegSize;
    tj3Destroy(h);
    return 0;
}

/* encode_interleaved_with_quant compresses using custom quantization tables
 * via jpeglib directly. Output is malloc'd; caller must free(). */
static int encode_interleaved_with_quant(
    const unsigned char *src_buf, int width, int pitch, int height,
    int pixel_format,
    int subsamp,
    const unsigned short *lum_table, const unsigned short *chrom_table,
    unsigned char **out_buf, unsigned long *out_size,
    char *err_buf, int err_buf_size)
{
    struct jpeg_compress_struct cinfo;
    struct jpeg_error_mgr jerr;
    cinfo.err = jpeg_std_error(&jerr);
    jpeg_create_compress(&cinfo);

    unsigned char *mem_buf = NULL;
    unsigned long mem_size = 0;
    jpeg_mem_dest(&cinfo, &mem_buf, &mem_size);

    cinfo.image_width = width;
    cinfo.image_height = height;

    switch (pixel_format) {
        case TJPF_RGBA:
            cinfo.input_components = 4;
            cinfo.in_color_space = JCS_EXT_RGBA;
            break;
        case TJPF_RGB:
            cinfo.input_components = 3;
            cinfo.in_color_space = JCS_RGB;
            break;
        case TJPF_GRAY:
            cinfo.input_components = 1;
            cinfo.in_color_space = JCS_GRAYSCALE;
            break;
        default:
            snprintf(err_buf, err_buf_size, "unsupported pixel format for quant table encode");
            jpeg_destroy_compress(&cinfo);
            return -1;
    }

    jpeg_set_defaults(&cinfo);

    switch (subsamp) {
        case TJSAMP_444:
            cinfo.comp_info[0].h_samp_factor = 1;
            cinfo.comp_info[0].v_samp_factor = 1;
            break;
        case TJSAMP_422:
            cinfo.comp_info[0].h_samp_factor = 2;
            cinfo.comp_info[0].v_samp_factor = 1;
            break;
        case TJSAMP_420:
            cinfo.comp_info[0].h_samp_factor = 2;
            cinfo.comp_info[0].v_samp_factor = 2;
            break;
        case TJSAMP_440:
            cinfo.comp_info[0].h_samp_factor = 1;
            cinfo.comp_info[0].v_samp_factor = 2;
            break;
        case TJSAMP_GRAY:
            break;
    }

    unsigned int lum32[64], chrom32[64];
    for (int i = 0; i < 64; i++) {
        lum32[i] = lum_table[i];
        chrom32[i] = chrom_table ? chrom_table[i] : lum_table[i];
    }
    jpeg_add_quant_table(&cinfo, 0, lum32, 100, FALSE);
    jpeg_add_quant_table(&cinfo, 1, chrom32, 100, FALSE);

    jpeg_start_compress(&cinfo, TRUE);

    int row_stride = pitch ? pitch : width * cinfo.input_components;
    while (cinfo.next_scanline < cinfo.image_height) {
        const unsigned char *row = src_buf + cinfo.next_scanline * row_stride;
        jpeg_write_scanlines(&cinfo, (JSAMPARRAY)&row, 1);
    }

    jpeg_finish_compress(&cinfo);
    jpeg_destroy_compress(&cinfo);

    *out_buf = mem_buf;
    *out_size = mem_size;
    return 0;
}

static int encode_yuv_planes(
    const unsigned char *y_buf, const unsigned char *cb_buf,
    const unsigned char *cr_buf,
    int y_stride, int c_stride,
    int width, int height, int subsamp,
    int quality,
    unsigned char **out_buf, unsigned long *out_size,
    char *err_buf, int err_buf_size)
{
    tjhandle h = tj3Init(TJINIT_COMPRESS);
    if (!h) {
        snprintf(err_buf, err_buf_size, "tj3Init failed");
        return -1;
    }

    if (tj3Set(h, TJPARAM_QUALITY, quality) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }
    if (tj3Set(h, TJPARAM_SUBSAMP, subsamp) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    const unsigned char *planes[3] = {y_buf, cb_buf, cr_buf};
    int strides[3] = {y_stride, c_stride, c_stride};

    *out_buf = NULL;
    size_t jpegSize = 0;

    if (tj3CompressFromYUVPlanes8(h, planes, width, strides, height,
                                  out_buf, &jpegSize) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    *out_size = (unsigned long)jpegSize;
    tj3Destroy(h);
    return 0;
}
