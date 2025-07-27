package main

import (
	"log"
	"os"

	"youtube/api"
	"youtube/downloader"

	"github.com/joho/godotenv"
)

func main() {
	// .envファイルからキーを読みこんで設定する。
	// YouTube Data APIのAPIキーとチャンネルIDを設定
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	channelId := os.Getenv("YOUR_CHANNEL_ID")

	// YouTube API クライアントを作成
	youtubeAPI := api.NewYouTubeAPI(apiKey, channelId)

	// プレイリストの動画一覧を取得
	items, err := youtubeAPI.GetPlaylistItems()
	if err != nil {
		log.Fatalf("Failed to get playlist items: %v", err)
	}

	// ダウンロードディレクトリを指定（環境変数から取得、なければデフォルト値）
	downloadDir := os.Getenv("DOWNLOAD_DIR")
	if downloadDir == "" {
		downloadDir = "./downloads" // デフォルトのダウンロードディレクトリ
	}

	// 動画ダウンローダーを作成
	videoDownloader := downloader.NewVideoDownloader(downloadDir)

	// 動画をダウンロード
	err = videoDownloader.DownloadVideos(items)
	if err != nil {
		log.Fatalf("Failed to download videos: %v", err)
	}
}
