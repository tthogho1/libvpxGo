package main

import (
	"flag"
	"log"
	"os"
	"strings"

	"youtube/api"
	"youtube/downloader"
	"youtube/lister"

	"github.com/joho/godotenv"
)

func main() {
	// コマンドラインフラグの定義
	var (
		mode      = flag.String("mode", "list", "Operation mode: list, download, formats")
		videoUrls = flag.String("urls", "", "Comma-separated video URLs (for list/formats mode)")
		videoIds  = flag.String("ids", "", "Comma-separated video IDs (for list/formats mode)")
	)
	flag.Parse()

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

	switch *mode {
	case "list":
		// 一覧表示モード
		videoLister := lister.NewVideoLister()
		
		if *videoIds != "" {
			// 指定されたビデオIDの一覧表示
			ids := strings.Split(*videoIds, ",")
			for i := range ids {
				ids[i] = strings.TrimSpace(ids[i])
			}
			err = videoLister.ListVideosByIds(ids)
		} else if *videoUrls != "" {
			// 指定されたURLからビデオIDを抽出して一覧表示
			urls := strings.Split(*videoUrls, ",")
			var ids []string
			for _, url := range urls {
				url = strings.TrimSpace(url)
				if strings.Contains(url, "v=") {
					parts := strings.Split(url, "v=")
					if len(parts) > 1 {
						videoId := strings.Split(parts[1], "&")[0]
						ids = append(ids, videoId)
					}
				}
			}
			err = videoLister.ListVideosByIds(ids)
		} else {
			// プレイリストの一覧表示
			err = videoLister.ListPlaylistVideos(items)
		}
		
		if err != nil {
			log.Fatalf("Failed to list videos: %v", err)
		}

	case "formats":
		// フォーマット表示モード
		videoLister := lister.NewVideoLister()
		
		if *videoUrls != "" {
			urls := strings.Split(*videoUrls, ",")
			for _, url := range urls {
				url = strings.TrimSpace(url)
				log.Printf("Listing formats for: %s", url)
				err = videoLister.ListVideoFormats(url)
				if err != nil {
					log.Printf("Failed to list formats for %s: %v", url, err)
				}
			}
		} else if *videoIds != "" {
			ids := strings.Split(*videoIds, ",")
			for _, id := range ids {
				id = strings.TrimSpace(id)
				videoUrl := "https://www.youtube.com/watch?v=" + id
				log.Printf("Listing formats for: %s", videoUrl)
				err = videoLister.ListVideoFormats(videoUrl)
				if err != nil {
					log.Printf("Failed to list formats for %s: %v", videoUrl, err)
				}
			}
		} else {
			log.Println("Please specify video URLs or IDs with -urls or -ids flag for formats mode")
		}

	case "download":
		// ダウンロードモード（従来の機能）
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

	default:
		log.Fatalf("Unknown mode: %s. Available modes: list, download, formats", *mode)
	}
}
