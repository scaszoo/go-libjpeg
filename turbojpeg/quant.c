#include "tj_helpers.h"
#include <jpeglib.h>

static int extract_quant_tables(
    const unsigned char *jpeg_buf, size_t jpeg_size,
    unsigned short *lum_table, unsigned short *chrom_table,
    int *has_chrom,
    char *err_buf, int err_buf_size)
{
    struct jpeg_decompress_struct dinfo;
    struct jpeg_error_mgr jerr;
    dinfo.err = jpeg_std_error(&jerr);
    jpeg_create_decompress(&dinfo);
    jpeg_mem_src(&dinfo, jpeg_buf, jpeg_size);

    if (jpeg_read_header(&dinfo, TRUE) != JPEG_HEADER_OK) {
        snprintf(err_buf, err_buf_size, "failed to read JPEG header");
        jpeg_destroy_decompress(&dinfo);
        return -1;
    }

    if (dinfo.quant_tbl_ptrs[0] != NULL) {
        for (int i = 0; i < 64; i++) {
            lum_table[i] = (unsigned short)dinfo.quant_tbl_ptrs[0]->quantval[i];
        }
    } else {
        snprintf(err_buf, err_buf_size, "no luminance quantization table found");
        jpeg_destroy_decompress(&dinfo);
        return -1;
    }

    *has_chrom = 0;
    if (dinfo.quant_tbl_ptrs[1] != NULL) {
        *has_chrom = 1;
        for (int i = 0; i < 64; i++) {
            chrom_table[i] = (unsigned short)dinfo.quant_tbl_ptrs[1]->quantval[i];
        }
    }

    jpeg_destroy_decompress(&dinfo);
    return 0;
}
