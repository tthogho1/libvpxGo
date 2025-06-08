package main

import (
	"fmt"
	"image"
	"log"
	"time"
	"unsafe"

	"github.com/xlab/libvpx-go/vpx"
	"gocv.io/x/gocv"
)


type VP8Encoder struct {
	ctx    *vpx.CodecCtx
	cfg    *vpx.CodecEncCfg
	width  int
	height int
}

func NewVP8Encoder(width, height int) (*VP8Encoder, error) {
	// VP8エンコーダーを初期化
	//iface := vpx.VPXCodecIfaceVP8Encoder
  iface := vpx.EncoderIfaceVP8()
	cfg := &vpx.CodecEncCfg{}
	if res := vpx.CodecEncConfigDefault(iface, cfg, 0); res != vpx.CodecOk {
		return nil, fmt.Errorf("VP8エンコーダー設定初期化失敗1: %v", res)
	}

	// 設定
	cfg.GW = 640             // 幅 (Width)
	cfg.GH = 480             // 高さ (Height)
	cfg.GTimebase.Num = 1    // タイムベースの分子 (例: 1/30秒 = 30fps)
	cfg.GTimebase.Den = 30   // タイムベースの分母
	cfg.RcTargetBitrate = 1000 // 目標ビットレート (kbps)
	cfg.GUsage = 1 // 1 = realtime mode for VP8 (see libvpx documentation)

	// 設定が正しく反映されたか、確認のためにプリントアウト
	fmt.Printf("Goエンコーダー設定 (cfg) - 幅: %d, 高さ: %d\n", cfg.GW, cfg.GH)
	fmt.Printf("Goエンコーダー設定 (cfg) - タイムベース (Num/Den): %d/%d\n", cfg.GTimebase.Num, cfg.GTimebase.Den)
	fmt.Printf("Goエンコーダー設定 (cfg) - 目標ビットレート: %d kbps\n", cfg.RcTargetBitrate)
	fmt.Printf("Goエンコーダー設定 (cfg) - Usage: %d\n", cfg.GUsage)
	fmt.Printf("Goエンコーダー設定 (cfg) - 全体: %+v\n", cfg)

	ctx := vpx.NewCodecCtx()
	if ctx == nil {
		fmt.Println("エラー: vpx.NewCodecCtx() が nil を返しました。")
		return nil, fmt.Errorf("vpx.NewCodecCtx() returned nil") // ここで処理を中断
	}
	fmt.Printf("CodecCtx が正常に割り当てられました: %p\n", ctx)
/*    ctx_size := unsafe.Sizeof(*ctx) // Go の CodecCtx のサイズ
    C.memset(unsafe.Pointer(ctx), 0, C.size_t(ctx_size)) // ゼロクリア実行

    fmt.Println("CodecCtx のメモリをゼロクリアしました。") */

	if res := vpx.CodecEncInitVer(ctx, iface, cfg, 0, vpx.EncoderABIVersion); res != vpx.CodecOk {
		return nil, fmt.Errorf("VP8エンコーダー初期化失敗2: %v", res)
	}
	
	fmt.Printf("VP8エンコーダー初期化成功: %v\n", ctx)
	return &VP8Encoder{
		ctx:    ctx,
		cfg:    cfg,
		width:  width,
		height: height,
	}, nil
}

func (e *VP8Encoder) Encode(mat gocv.Mat) ([]byte, error) {
	// GoCVのMatからRGBデータを取得
	img, err := mat.ToImage()
	if err != nil {
		return nil, fmt.Errorf("Mat変換エラー: %v", err)
	}
	
	// RGBからYUV420に変換
	yuvData := rgbToYUV420(img, e.width, e.height)
	
	// VP8エンコード用のイメージを作成
	vpxImg := vpx.ImageAlloc(nil, vpx.ImageFormatI420, uint32(e.width), uint32(e.height), 1)
	if vpxImg == nil {
		return nil, fmt.Errorf("vpx image allocation failed")
	}
	// Set Y, U, V planes


	copy((*[1 << 30]byte)(unsafe.Pointer(vpxImg.Planes[0]))[:len(yuvData[0]):len(yuvData[0])], yuvData[0])
	copy((*[1 << 30]byte)(unsafe.Pointer(vpxImg.Planes[1]))[:len(yuvData[1]):len(yuvData[1])], yuvData[1])
	copy((*[1 << 30]byte)(unsafe.Pointer(vpxImg.Planes[2]))[:len(yuvData[2]):len(yuvData[2])], yuvData[2])
	
	// エンコード実行
	deadline := uint64(time.Now().UnixNano() / 1000) // マイクロ秒
	if res := vpx.CodecEncode(e.ctx, vpxImg, 0, 1, 0, uint(deadline)); res != vpx.CodecOk {
		return nil, fmt.Errorf("VP8エンコードエラー: %v", res)
	}
	
	// エンコード結果を取得
	var encoded []byte
	var iter vpx.CodecIter = nil
	for {
		pkt := vpx.CodecGetCxData(e.ctx, &iter)
		if pkt == nil {
			break
		}
		pkt.Deref() // 必須
		if pkt.Kind == vpx.CodecCxFramePkt {
	
			frame := (*vpx.FixedBuf)(unsafe.Pointer(pkt))

			frameSz := frame.Sz
			frameData := unsafe.Slice((*byte)(frame.Buf), frameSz)
			encoded = append(encoded, frameData...)
		}
	}
	
	return encoded, nil
}

func (e *VP8Encoder) Close() {
	if e.ctx != nil {
		vpx.CodecDestroy(e.ctx)
	}
}

// RGBからYUV420に変換
func rgbToYUV420(img image.Image, width, height int) [3][]byte {
	bounds := img.Bounds()
	yData := make([]byte, width*height)
	uData := make([]byte, width*height/4)
	vData := make([]byte, width*height/4)
	
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			r8, g8, b8 := uint8(r>>8), uint8(g>>8), uint8(b>>8)
			
			// RGB to YUV変換
			Y := uint8((66*int(r8) + 129*int(g8) + 25*int(b8) + 128) >> 8 + 16)
			U := uint8((-38*int(r8) - 74*int(g8) + 112*int(b8) + 128) >> 8 + 128)
			V := uint8((112*int(r8) - 94*int(g8) - 18*int(b8) + 128) >> 8 + 128)
			
			yData[y*width+x] = Y
			
			// UV は 2x2 サブサンプリング
			if y%2 == 0 && x%2 == 0 {
				uvIndex := (y/2)*(width/2) + (x/2)
				uData[uvIndex] = U
				vData[uvIndex] = V
			}
		}
	}
	
	return [3][]byte{yData, uData, vData}
}

// 使用例
func main() {
	// GoCV初期化
	webcam, err := gocv.OpenVideoCapture(0)
	if err != nil {
		log.Fatal(err)
	}
	defer webcam.Close()
	
	// VP8エンコーダー初期化
	encoder, err := NewVP8Encoder(640, 480)
	if err != nil {
		log.Fatal(err)
	}
	defer encoder.Close()
	
	mat := gocv.NewMat()
	defer mat.Close()
	
	// 連続エンコード
	for {
		if ok := webcam.Read(&mat); !ok {
			break
		}
		
		if mat.Empty() {
			continue
		}
		
		// VP8エンコード
		encoded, err := encoder.Encode(mat)
		if err != nil {
			log.Printf("エンコードエラー: %v", err)
			continue
		}
		
		// WebRTCに送信 (ここでWebRTCライブラリを使用)
		fmt.Printf("VP8フレーム生成: %d bytes\n", len(encoded))
		
		// 30fps制御
		time.Sleep(33 * time.Millisecond)
	}
}