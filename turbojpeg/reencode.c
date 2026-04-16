#include "tj_helpers.h"
#include <jpeglib.h>

static int reencode(
    const unsigned char *jpeg_buf, size_t jpeg_size,
    unsigned char **out_buf, unsigned long *out_size,
    int quality, int subsamp,
    const unsigned short *lum_quant, const unsigned short *chrom_quant,
    char *err_buf, int err_buf_size)
{
    tjhandle dec = NULL;
    tjhandle enc = NULL;
    unsigned char *rgb = NULL;
    int ret = -1;
    int use_quant = (lum_quant != NULL);

    /* Phase 1: Decompress to RGB */
    dec = tj3Init(TJINIT_DECOMPRESS);
    if (!dec) {
        snprintf(err_buf, err_buf_size, "tj3Init(DECOMPRESS) failed");
        goto cleanup;
    }

    if (tj3DecompressHeader(dec, jpeg_buf, jpeg_size) < 0) {
        snprintf(err_buf, err_buf_size, "decompress header: %s", tj3GetErrorStr(dec));
        goto cleanup;
    }

    int w = tj3Get(dec, TJPARAM_JPEGWIDTH);
    int h = tj3Get(dec, TJPARAM_JPEGHEIGHT);
    int pitch = w * tjPixelSize[TJPF_RGB];

    rgb = (unsigned char *)tj3Alloc((size_t)pitch * (size_t)h);
    if (!rgb) {
        snprintf(err_buf, err_buf_size, "failed to allocate RGB buffer");
        goto cleanup;
    }

    if (tj3Decompress8(dec, jpeg_buf, jpeg_size, rgb, pitch, TJPF_RGB) < 0) {
        snprintf(err_buf, err_buf_size, "decompress: %s", tj3GetErrorStr(dec));
        goto cleanup;
    }

    /* Phase 2: Compress from RGB */
    if (use_quant) {
        /* Custom quant tables — use jpeglib directly */
        struct jpeg_compress_struct cinfo;
        struct jpeg_error_mgr jerr;
        cinfo.err = jpeg_std_error(&jerr);
        jpeg_create_compress(&cinfo);

        unsigned char *mem_buf = NULL;
        unsigned long mem_size = 0;
        jpeg_mem_dest(&cinfo, &mem_buf, &mem_size);

        cinfo.image_width = w;
        cinfo.image_height = h;
        cinfo.input_components = 3;
        cinfo.in_color_space = JCS_RGB;
        jpeg_set_defaults(&cinfo);

        switch (subsamp) {
            case TJSAMP_444:
                cinfo.comp_info[0].h_samp_factor = 1;
                cinfo.comp_info[0].v_samp_factor = 1; break;
            case TJSAMP_422:
                cinfo.comp_info[0].h_samp_factor = 2;
                cinfo.comp_info[0].v_samp_factor = 1; break;
            case TJSAMP_420:
                cinfo.comp_info[0].h_samp_factor = 2;
                cinfo.comp_info[0].v_samp_factor = 2; break;
            case TJSAMP_440:
                cinfo.comp_info[0].h_samp_factor = 1;
                cinfo.comp_info[0].v_samp_factor = 2; break;
        }

        unsigned int lum32[64], chrom32[64];
        for (int i = 0; i < 64; i++) {
            lum32[i] = lum_quant[i];
            chrom32[i] = chrom_quant ? chrom_quant[i] : lum_quant[i];
        }
        jpeg_add_quant_table(&cinfo, 0, lum32, 100, FALSE);
        jpeg_add_quant_table(&cinfo, 1, chrom32, 100, FALSE);

        jpeg_start_compress(&cinfo, TRUE);
        while (cinfo.next_scanline < cinfo.image_height) {
            const unsigned char *row = rgb + cinfo.next_scanline * pitch;
            jpeg_write_scanlines(&cinfo, (JSAMPARRAY)&row, 1);
        }
        jpeg_finish_compress(&cinfo);
        jpeg_destroy_compress(&cinfo);

        *out_buf = mem_buf;
        *out_size = mem_size;
    } else {
        /* Standard quality-based — use TurboJPEG */
        enc = tj3Init(TJINIT_COMPRESS);
        if (!enc) {
            snprintf(err_buf, err_buf_size, "tj3Init(COMPRESS) failed");
            goto cleanup;
        }

        if (tj3Set(enc, TJPARAM_QUALITY, quality) < 0) {
            snprintf(err_buf, err_buf_size, "set quality: %s", tj3GetErrorStr(enc));
            goto cleanup;
        }
        if (tj3Set(enc, TJPARAM_SUBSAMP, subsamp) < 0) {
            snprintf(err_buf, err_buf_size, "set subsamp: %s", tj3GetErrorStr(enc));
            goto cleanup;
        }

        *out_buf = NULL;
        size_t jpegSize = 0;

        if (tj3Compress8(enc, rgb, w, pitch, h, TJPF_RGB,
                         out_buf, &jpegSize) < 0) {
            snprintf(err_buf, err_buf_size, "compress: %s", tj3GetErrorStr(enc));
            goto cleanup;
        }
        *out_size = (unsigned long)jpegSize;
    }

    ret = 0;

cleanup:
    if (rgb) tj3Free(rgb);
    if (dec) tj3Destroy(dec);
    if (enc) tj3Destroy(enc);
    return ret;
}
