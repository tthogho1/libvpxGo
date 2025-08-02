package downloader

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"youtube/types"

	"github.com/kkdai/youtube/v2"
)

// VideoDownloader handles YouTube video downloads
type VideoDownloader struct {
	client     youtube.Client
	outputDir  string
}

// NewVideoDownloader creates a new VideoDownloader instance
func NewVideoDownloader(outputDir string) *VideoDownloader {
	return &VideoDownloader{
		client:    youtube.Client{},
		outputDir: outputDir,
	}
}

// DownloadVideos downloads all videos from the provided playlist items
func (d *VideoDownloader) DownloadVideos(items []types.PlaylistItem) error {
	// 出力ディレクトリが存在しない場合は作成
	if err := os.MkdirAll(d.outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %v", err)
	}

	for _, item := range items {
		fmt.Printf("Title: %s, VideoId: %s\n", item.Title, item.VideoId)
		
		// download動画のURLを生成
		videoUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.VideoId)
		fmt.Println("Download URL:", videoUrl)
		
		if err := d.downloadSingleVideo(videoUrl, item); err != nil {
			log.Printf("Failed to download video %s: %v", item.Title, err)
			continue
		}
		
		fmt.Println("Downloaded:", item.Title)
	}
	return nil
}

// downloadSingleVideo downloads a single video
func (d *VideoDownloader) downloadSingleVideo(videoUrl string, item types.PlaylistItem) error {
	video, err := d.client.GetVideo(videoUrl)
	if err != nil {
		return fmt.Errorf("failed to get video: %v", err)
	}

	// 音声付きの適切な形式を取得
	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return fmt.Errorf("no formats with audio found")
	}

	// または品質で絞り込む場合
	var format *youtube.Format
	for _, f := range formats {
		if f.MimeType == "video/mp4" && 
			 (f.QualityLabel == "720p" || f.QualityLabel == "480p") {
			format = &f
			break
		} else {
			fmt.Printf("Skipping format: %s, Quality: %s\n", f.MimeType, f.QualityLabel)
		}
	}

	if format == nil {
		format = &formats[0]
	}

	// 音声しか取れない。
	stream, _, err := d.client.GetStream(video, format)
	if err != nil {
		return fmt.Errorf("failed to get stream: %v", err)
	}
	defer stream.Close()
	
	// 出力ファイルパスを指定ディレクトリに設定
	fileName := item.VideoId + ".mp4"
	filePath := filepath.Join(d.outputDir, fileName)
	
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	_, err = io.Copy(file, stream)
	if err != nil {
		return fmt.Errorf("failed to copy stream to file: %v", err)
	}

	fmt.Printf("Saved to: %s\n", filePath)
	return nil
}
