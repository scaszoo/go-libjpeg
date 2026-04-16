#include "tj_helpers.h"

static int decode_header(
    const unsigned char *jpeg_buf, size_t jpeg_size,
    int *out_width, int *out_height, int *out_subsamp,
    char *err_buf, int err_buf_size)
{
    tjhandle h = tj3Init(TJINIT_DECOMPRESS);
    if (!h) {
        snprintf(err_buf, err_buf_size, "tj3Init failed");
        return -1;
    }

    if (tj3DecompressHeader(h, jpeg_buf, jpeg_size) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    *out_width = tj3Get(h, TJPARAM_JPEGWIDTH);
    *out_height = tj3Get(h, TJPARAM_JPEGHEIGHT);
    *out_subsamp = tj3Get(h, TJPARAM_SUBSAMP);

    tj3Destroy(h);
    return 0;
}

static int decode_interleaved(
    const unsigned char *jpeg_buf, size_t jpeg_size,
    unsigned char *dst_buf,
    int *out_width, int *out_height, int *out_stride,
    int pixel_format,
    int scale_num, int scale_denom,
    char *err_buf, int err_buf_size)
{
    tjhandle h = tj3Init(TJINIT_DECOMPRESS);
    if (!h) {
        snprintf(err_buf, err_buf_size, "tj3Init failed");
        return -1;
    }

    if (tj3DecompressHeader(h, jpeg_buf, jpeg_size) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    if (scale_num > 0 && scale_denom > 0) {
        tjscalingfactor sf = {scale_num, scale_denom};
        if (tj3SetScalingFactor(h, sf) < 0) {
            snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
            tj3Destroy(h);
            return -1;
        }
    }

    int w = tj3Get(h, TJPARAM_JPEGWIDTH);
    int ht = tj3Get(h, TJPARAM_JPEGHEIGHT);
    if (scale_num > 0 && scale_denom > 0) {
        tjscalingfactor sf = {scale_num, scale_denom};
        w = TJSCALED(w, sf);
        ht = TJSCALED(ht, sf);
    }

    *out_width = w;
    *out_height = ht;
    *out_stride = w * tjPixelSize[pixel_format];

    if (tj3Decompress8(h, jpeg_buf, jpeg_size,
                       dst_buf, *out_stride, pixel_format) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    tj3Destroy(h);
    return 0;
}

static int decode_header_dims(
    const unsigned char *jpeg_buf, size_t jpeg_size,
    int pixel_format,
    int scale_num, int scale_denom,
    int *out_width, int *out_height, int *out_stride,
    char *err_buf, int err_buf_size)
{
    tjhandle h = tj3Init(TJINIT_DECOMPRESS);
    if (!h) {
        snprintf(err_buf, err_buf_size, "tj3Init failed");
        return -1;
    }

    if (tj3DecompressHeader(h, jpeg_buf, jpeg_size) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    int w = tj3Get(h, TJPARAM_JPEGWIDTH);
    int ht = tj3Get(h, TJPARAM_JPEGHEIGHT);
    if (scale_num > 0 && scale_denom > 0) {
        tjscalingfactor sf = {scale_num, scale_denom};
        w = TJSCALED(w, sf);
        ht = TJSCALED(ht, sf);
    }

    *out_width = w;
    *out_height = ht;
    *out_stride = w * tjPixelSize[pixel_format];

    tj3Destroy(h);
    return 0;
}

static int decode_yuv_planes(
    const unsigned char *jpeg_buf, size_t jpeg_size,
    unsigned char *y_buf, unsigned char *cb_buf, unsigned char *cr_buf,
    int *y_stride, int *c_stride,
    int *out_width, int *out_height, int *out_subsamp,
    int scale_num, int scale_denom,
    char *err_buf, int err_buf_size)
{
    tjhandle h = tj3Init(TJINIT_DECOMPRESS);
    if (!h) {
        snprintf(err_buf, err_buf_size, "tj3Init failed");
        return -1;
    }

    if (tj3DecompressHeader(h, jpeg_buf, jpeg_size) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    if (scale_num > 0 && scale_denom > 0) {
        tjscalingfactor sf = {scale_num, scale_denom};
        if (tj3SetScalingFactor(h, sf) < 0) {
            snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
            tj3Destroy(h);
            return -1;
        }
    }

    int w = tj3Get(h, TJPARAM_JPEGWIDTH);
    int ht = tj3Get(h, TJPARAM_JPEGHEIGHT);
    int subsamp = tj3Get(h, TJPARAM_SUBSAMP);

    if (scale_num > 0 && scale_denom > 0) {
        tjscalingfactor sf = {scale_num, scale_denom};
        w = TJSCALED(w, sf);
        ht = TJSCALED(ht, sf);
    }

    *out_width = w;
    *out_height = ht;
    *out_subsamp = subsamp;

    *y_stride = tj3YUVPlaneWidth(0, w, subsamp);
    *c_stride = (subsamp == TJSAMP_GRAY) ? 0 : tj3YUVPlaneWidth(1, w, subsamp);

    unsigned char *planes[3] = {y_buf, cb_buf, cr_buf};
    int strides[3] = {*y_stride, *c_stride, *c_stride};

    if (tj3DecompressToYUVPlanes8(h, jpeg_buf, jpeg_size, planes, strides) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    tj3Destroy(h);
    return 0;
}

static int decode_yuv_planes_dims(
    const unsigned char *jpeg_buf, size_t jpeg_size,
    int scale_num, int scale_denom,
    int *out_width, int *out_height, int *out_subsamp,
    int *y_stride, int *c_stride,
    size_t *y_size, size_t *cb_size, size_t *cr_size,
    char *err_buf, int err_buf_size)
{
    tjhandle h = tj3Init(TJINIT_DECOMPRESS);
    if (!h) {
        snprintf(err_buf, err_buf_size, "tj3Init failed");
        return -1;
    }

    if (tj3DecompressHeader(h, jpeg_buf, jpeg_size) < 0) {
        snprintf(err_buf, err_buf_size, "%s", tj3GetErrorStr(h));
        tj3Destroy(h);
        return -1;
    }

    int w = tj3Get(h, TJPARAM_JPEGWIDTH);
    int ht = tj3Get(h, TJPARAM_JPEGHEIGHT);
    int subsamp = tj3Get(h, TJPARAM_SUBSAMP);

    if (scale_num > 0 && scale_denom > 0) {
        tjscalingfactor sf = {scale_num, scale_denom};
        w = TJSCALED(w, sf);
        ht = TJSCALED(ht, sf);
    }

    *out_width = w;
    *out_height = ht;
    *out_subsamp = subsamp;

    *y_stride = tj3YUVPlaneWidth(0, w, subsamp);
    *c_stride = (subsamp == TJSAMP_GRAY) ? 0 : tj3YUVPlaneWidth(1, w, subsamp);

    *y_size = (size_t)tj3YUVPlaneSize(0, w, *y_stride, ht, subsamp);
    *cb_size = (subsamp == TJSAMP_GRAY) ? 0 :
        (size_t)tj3YUVPlaneSize(1, w, *c_stride, ht, subsamp);
    *cr_size = (subsamp == TJSAMP_GRAY) ? 0 :
        (size_t)tj3YUVPlaneSize(2, w, *c_stride, ht, subsamp);

    tj3Destroy(h);
    return 0;
}

