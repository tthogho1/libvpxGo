#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#include <vpx/vpx_encoder.h>
#include <vpx/vp8cx.h> // VP8 エンコーダーのインターフェース用

#define WIDTH  640
#define HEIGHT 480
#define FPS    30
#define BITRATE_KBPS 1000

int main() {
    vpx_codec_iface_t *iface = vpx_codec_vp8_cx(); // VP8 エンコーダーインターフェースを取得
    if (iface == NULL) {
        fprintf(stderr, "Failed to get VP8 encoder interface.\n");
        return EXIT_FAILURE;
    }

    vpx_codec_ctx_t codec;
    vpx_codec_enc_cfg_t cfg;

    // デフォルト設定の取得
    vpx_codec_err_t res = vpx_codec_enc_config_default(iface, &cfg, 0);
    if (res != VPX_CODEC_OK) {
        fprintf(stderr, "Failed to get default encoder config: %s\n", vpx_codec_err_to_string(res));
        return EXIT_FAILURE;
    }

    // 必須設定値のセット
    cfg.g_w = WIDTH;
    cfg.g_h = HEIGHT;
    cfg.g_timebase.num = 1;
    cfg.g_timebase.den = FPS;
    cfg.rc_target_bitrate = BITRATE_KBPS;
    cfg.g_usage =  VPX_DL_REALTIME; // リアルタイムエンコードの例

    printf("Encoder config: Width=%d, Height=%d, Timebase=%d/%d, Bitrate=%d kbps\n",
           cfg.g_w, cfg.g_h, cfg.g_timebase.num, cfg.g_timebase.den, cfg.rc_target_bitrate);

    // エンコーダーの初期化
    res = vpx_codec_enc_init(&codec, iface, &cfg, 0); // vpx_codec_enc_init は vpx_codec_enc_init_ver の簡略版
    if (res != VPX_CODEC_OK) {
        fprintf(stderr, "Failed to initialize encoder: %s\n", vpx_codec_err_to_string(res));
        // エラーコードが VPX_CODEC_MEM_ERROR (3) の場合、ここで出るはず
        if (res == VPX_CODEC_MEM_ERROR) {
            fprintf(stderr, "Error is VPX_CODEC_MEM_ERROR (3). This indicates a memory allocation failure.\n");
        }
        return EXIT_FAILURE;
    }

    printf("Encoder initialized successfully!\n");

    // ここでエンコード処理 (ダミーなので省略)
    // 実際のアプリケーションでは vpx_img_alloc で画像を作成し、vpx_codec_encode でエンコードします。

    // エンコーダーの破棄
    vpx_codec_destroy(&codec);
    printf("Encoder destroyed.\n");

    return EXIT_SUCCESS;
}