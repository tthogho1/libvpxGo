package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"youtube/api"
	"youtube/downloader"
	"youtube/dynamo"
	"youtube/lister"

	"github.com/joho/godotenv"
)

func main() {
	// コマンドラインフラグの定義
	var (
		mode      = flag.String("mode", "list", "Operation mode: list, download, formats, transcribe-status, mark-transcribed, list-untranscribed, list-transcribed, test-db")
		videoUrls = flag.String("urls", "", "Comma-separated video URLs (for list/formats mode)")
		videoIds  = flag.String("ids", "", "Comma-separated video IDs (for list/formats mode)")
		saveToDB  = flag.Bool("save-db", false, "Save video information to DynamoDB")
		tableName = flag.String("table", "youtube_videos", "DynamoDB table name")
		transcribeStatus = flag.Bool("transcribed", false, "Set transcribe status (use with -mode=transcribe-status and -ids)")
	)
	flag.Parse()

	// .envファイルからキーを読みこんで設定する。
	// YouTube Data APIのAPIキーとチャンネルIDを設定
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}
	
	// 環境変数の確認（デバッグ用）
	log.Printf("DEBUG: AWS_ACCESS_KEY_ID loaded: %t", os.Getenv("AWS_ACCESS_KEY_ID") != "")
	log.Printf("DEBUG: AWS_SECRET_ACCESS_KEY loaded: %t", os.Getenv("AWS_SECRET_ACCESS_KEY") != "")
	log.Printf("DEBUG: AWS_REGION: %s", os.Getenv("AWS_REGION"))
	
	apiKey := os.Getenv("YOUTUBE_API_KEY")
	channelId := os.Getenv("YOUR_CHANNEL_ID")

	switch *mode {
	case "list":
		// 一覧表示モード
		videoLister, err := lister.NewVideoLister(*saveToDB, *tableName)
		if err != nil {
			log.Fatalf("Failed to create video lister: %v", err)
		}
		
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
			// プレイリストの一覧表示（このタイミングでのみYouTube APIを呼ぶ）
			youtubeAPI := api.NewYouTubeAPI(apiKey, channelId)
			items, ierr := youtubeAPI.GetPlaylistItems()
			if ierr != nil {
				log.Fatalf("Failed to get playlist items: %v", ierr)
			}
			err = videoLister.ListPlaylistVideos(items)
		}
		
		if err != nil {
			log.Fatalf("Failed to list videos: %v", err)
		}

	case "formats":
		// フォーマット表示モード（DynamoDB保存なし）
		videoLister, err := lister.NewVideoLister(false, "")
		if err != nil {
			log.Fatalf("Failed to create video lister: %v", err)
		}
		
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

		// プレイリストの動画一覧を取得（downloadモード時にのみYouTube APIを呼ぶ）
		youtubeAPI := api.NewYouTubeAPI(apiKey, channelId)
		items, ierr := youtubeAPI.GetPlaylistItems()
		if ierr != nil {
			log.Fatalf("Failed to get playlist items: %v", ierr)
		}

		// 動画をダウンロード
		err = videoDownloader.DownloadVideos(items)
		if err != nil {
			log.Fatalf("Failed to download videos: %v", err)
		}

	case "transcribe-status":
		// Transcribeステータス更新モード
		if *videoIds == "" {
			log.Fatalf("Please specify video IDs with -ids flag for transcribe-status mode")
		}

		// DynamoDBクライアントを作成
		dynamoClient, err := dynamo.NewDynamoDBClient(*tableName)
		if err != nil {
			log.Fatalf("Failed to create DynamoDB client: %v", err)
		}

		// 指定された動画IDのtranscribeステータスを更新
		ids := strings.Split(*videoIds, ",")
		ctx := context.Background()
		
		for _, id := range ids {
			id = strings.TrimSpace(id)
			err = dynamoClient.UpdateTranscribeStatus(ctx, id, *transcribeStatus)
			if err != nil {
				log.Printf("Failed to update transcribe status for %s: %v", id, err)
			}
		}

	case "mark-transcribed":
		// 指定IDのTranscribedをtrueに単純設定
		if *videoIds == "" {
			log.Fatalf("Please specify video IDs with -ids flag for mark-transcribed mode")
		}

		// DynamoDBクライアントを作成
		dynamoClient, err := dynamo.NewDynamoDBClient(*tableName)
		if err != nil {
			log.Fatalf("Failed to create DynamoDB client: %v", err)
		}

		ids := strings.Split(*videoIds, ",")
		ctx := context.Background()
		for _, id := range ids {
			id = strings.TrimSpace(id)
			if id == "" { continue }
			if err := dynamoClient.UpdateTranscribeStatus(ctx, id, true); err != nil {
				log.Printf("Failed to mark transcribed for %s: %v", id, err)
			} else {
				fmt.Printf("Marked transcribed: %s\n", id)
			}
		}

	case "list-untranscribed":
		// Transcribe未実施動画の一覧表示
		dynamoClient, err := dynamo.NewDynamoDBClient(*tableName)
		if err != nil {
			log.Fatalf("Failed to create DynamoDB client: %v", err)
		}

		ctx := context.Background()
		videos, err := dynamoClient.GetUntranscribedVideos(ctx)
		if err != nil {
			log.Fatalf("Failed to get untranscribed videos: %v", err)
		}

		fmt.Printf("Found %d untranscribed videos:\n\n", len(videos))
		for i, video := range videos {
			fmt.Printf("%d. Title: %s\n", i+1, video.Title)
			fmt.Printf("   Video ID: %s\n", video.VideoID)
			fmt.Printf("   URL: %s\n", video.URL)
			fmt.Printf("   Author: %s\n", video.Author)
			fmt.Printf("   Duration: %s\n", video.Duration)
			fmt.Printf("   Views: %d\n", video.Views)
			fmt.Printf("   Transcribed: %t\n", video.Transcribed)
			fmt.Println()
		}

	case "list-transcribed":
		// Transcribe実施済み動画の一覧表示
		dynamoClient, err := dynamo.NewDynamoDBClient(*tableName)
		if err != nil {
			log.Fatalf("Failed to create DynamoDB client: %v", err)
		}

		ctx := context.Background()
		videos, err := dynamoClient.GetTranscribedVideos(ctx)
		if err != nil {
			log.Fatalf("Failed to get transcribed videos: %v", err)
		}

		fmt.Printf("Found %d transcribed videos:\n\n", len(videos))
		for i, video := range videos {
			fmt.Printf("%d. Title: %s\n", i+1, video.Title)
			fmt.Printf("   Video ID: %s\n", video.VideoID)
			fmt.Printf("   URL: %s\n", video.URL)
			fmt.Printf("   Author: %s\n", video.Author)
			fmt.Printf("   Duration: %s\n", video.Duration)
			fmt.Printf("   Views: %d\n", video.Views)
			fmt.Printf("   Transcribed: %t\n", video.Transcribed)
			fmt.Println()
		}

	case "test-db":
		// DynamoDB接続テスト
		fmt.Println("Testing DynamoDB connection...")
		dynamoClient, err := dynamo.NewDynamoDBClient(*tableName)
		if err != nil {
			log.Fatalf("Failed to create DynamoDB client: %v", err)
		}

		ctx := context.Background()
		if err := dynamoClient.CreateTableIfNotExists(ctx); err != nil {
			log.Fatalf("Failed to create table: %v", err)
		}

		if err := dynamoClient.TestConnection(ctx); err != nil {
			log.Fatalf("DynamoDB connection test failed: %v", err)
		}

		fmt.Println("🎉 DynamoDB connection test passed!")

	default:
		log.Fatalf("Unknown mode: %s. Available modes: list, download, formats, transcribe-status, mark-transcribed, list-untranscribed, list-transcribed, test-db", *mode)
	}
}
