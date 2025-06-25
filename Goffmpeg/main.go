package main

import (
	"os"

	ffmpeg "github.com/u2takey/ffmpeg-go"
)

func main() {
    // set ffmpeg cmd path if not in PATH using WithBinary option
    oldPath := os.Getenv("PATH")
    os.Setenv("PATH", oldPath+";C:\\ffmpeg\\bin")

    err := ffmpeg.Input("TEST.mp4").
        Output("output.mp3").
        OverWriteOutput().
        Run()
    if err != nil {
        panic(err)
    }

    // TEST.mp4 から画像(jpeg)を抽出
    // 例: 1秒ごとにフレームを抽出し、output_%03d.jpg で保存
    err = ffmpeg.Input("TEST.mp4").
        Output("output_%03d.jpg", ffmpeg.KwArgs{"vf": "fps=1"}).
        OverWriteOutput().
        Run()
    if err != nil {
        panic(err)
    }
}
