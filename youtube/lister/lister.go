package lister

import (
	"context"
	"fmt"
	"log"

	"youtube/dynamo"
	"youtube/types"

	"github.com/kkdai/youtube/v2"
)

// VideoLister handles YouTube video list operations
type VideoLister struct {
	client       youtube.Client
	dynamoClient *dynamo.DynamoDBClient
	saveToDB     bool
}

// NewVideoLister creates a new VideoLister instance
func NewVideoLister(saveToDB bool, tableName string) (*VideoLister, error) {
	var dynamoClient *dynamo.DynamoDBClient
	var err error

	if saveToDB {
		dynamoClient, err = dynamo.NewDynamoDBClient(tableName)
		if err != nil {
			return nil, fmt.Errorf("failed to create DynamoDB client: %v", err)
		}

		// テーブルを作成（存在しない場合）
		ctx := context.Background()
		if err := dynamoClient.CreateTableIfNotExists(ctx); err != nil {
			return nil, fmt.Errorf("failed to create table: %v", err)
		}
	}

	return &VideoLister{
		client:       youtube.Client{},
		dynamoClient: dynamoClient,
		saveToDB:     saveToDB,
	}, nil
}

// GetVideoInfo gets information about a single video
func (l *VideoLister) GetVideoInfo(videoUrl string) (*youtube.Video, error) {
	video, err := l.client.GetVideo(videoUrl)
	if err != nil {
		return nil, fmt.Errorf("failed to get video: %v", err)
	}
	return video, nil
}

// ListVideoFormats lists all available formats for a video
func (l *VideoLister) ListVideoFormats(videoUrl string) error {
	video, err := l.GetVideoInfo(videoUrl)
	if err != nil {
		return err
	}

	fmt.Printf("Video Title: %s\n", video.Title)
	fmt.Printf("Video ID: %s\n", video.ID)
	fmt.Printf("Duration: %s\n", video.Duration)
	fmt.Printf("Author: %s\n", video.Author)
	fmt.Printf("View Count: %d\n", video.Views)
	fmt.Printf("Description: %s\n", video.Description)
	fmt.Println("\n--- Available Formats ---")

	// すべてのフォーマットを表示
	for i, format := range video.Formats {
		fmt.Printf("Format %d:\n", i+1)
		fmt.Printf("  MIME Type: %s\n", format.MimeType)
		fmt.Printf("  Quality: %s\n", format.QualityLabel)
		fmt.Printf("  Bitrate: %d\n", format.Bitrate)
		fmt.Printf("  Audio Channels: %d\n", format.AudioChannels)
		fmt.Printf("  Audio Sample Rate: %s\n", format.AudioSampleRate)
		fmt.Printf("  Video Resolution: %dx%d\n", format.Width, format.Height)
		fmt.Printf("  FPS: %d\n", format.FPS)
		fmt.Printf("  Content Length: %d bytes\n", format.ContentLength)
		fmt.Println("  ---")
	}

	return nil
}

// ListPlaylistVideos lists all videos from the provided playlist items
func (l *VideoLister) ListPlaylistVideos(items []types.PlaylistItem) error {
	fmt.Printf("Found %d videos in playlist:\n\n", len(items))

	var videoRecords []dynamo.VideoRecord
	ctx := context.Background()

	for i, item := range items {
		fmt.Printf("%d. Title: %s\n", i+1, item.Title)
		fmt.Printf("   Video ID: %s\n", item.VideoId)
		fmt.Printf("   URL: https://www.youtube.com/watch?v=%s\n", item.VideoId)
		
		// 動画の詳細情報を取得
		videoUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.VideoId)
		video, err := l.GetVideoInfo(videoUrl)
		if err != nil {
			log.Printf("Failed to get video info for %s: %v", item.Title, err)
			fmt.Printf("   Duration: Unknown\n")
			fmt.Printf("   Views: Unknown\n")
			fmt.Printf("   Author: Unknown\n")
			
			// DynamoDBに保存する場合はエラー情報でも記録
			if l.saveToDB {
				videoRecord := dynamo.ConvertPlaylistItemToVideoRecord(item, "Unknown", 0, "Unknown", "Failed to fetch details")
				videoRecords = append(videoRecords, videoRecord)
			}
		} else {
			fmt.Printf("   Duration: %s\n", video.Duration)
			fmt.Printf("   Views: %d\n", video.Views)
			fmt.Printf("   Author: %s\n", video.Author)
			
			// DynamoDBに保存する場合はVideoRecordを作成
			if l.saveToDB {
				description := video.Description
				if len(description) > 500 {
					description = description[:500] + "..."
				}
				videoRecord := dynamo.ConvertPlaylistItemToVideoRecord(
					item, 
					video.Duration.String(), 
					int64(video.Views), 
					video.Author, 
					description,
				)
				videoRecords = append(videoRecords, videoRecord)
			}
		}
		fmt.Println()
	}

	// DynamoDBに一括保存（改良版のリトライ機能付き）
	if l.saveToDB && len(videoRecords) > 0 {
		fmt.Printf("Saving %d videos to DynamoDB with retry logic...\n", len(videoRecords))
		if err := l.dynamoClient.SaveVideosWithRetry(ctx, videoRecords); err != nil {
			log.Printf("Failed to save videos to DynamoDB: %v", err)
			// フォールバックとして標準版を試行
			fmt.Printf("Attempting fallback save without advanced retry...\n")
			if fallbackErr := l.dynamoClient.SaveVideos(ctx, videoRecords); fallbackErr != nil {
				log.Printf("Fallback save also failed: %v", fallbackErr)
			} else {
				fmt.Printf("Fallback save successful!\n")
			}
		} else {
			fmt.Printf("Successfully saved all videos to DynamoDB!\n")
		}
	}

	return nil
}

// ListVideosByIds lists videos by their IDs
func (l *VideoLister) ListVideosByIds(videoIds []string) error {
	fmt.Printf("Listing %d videos:\n\n", len(videoIds))

	var videoRecords []dynamo.VideoRecord
	ctx := context.Background()

	for i, videoId := range videoIds {
		videoUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoId)
		video, err := l.GetVideoInfo(videoUrl)
		if err != nil {
			log.Printf("Failed to get video info for %s: %v", videoId, err)
			continue
		}

		fmt.Printf("%d. Title: %s\n", i+1, video.Title)
		fmt.Printf("   Video ID: %s\n", video.ID)
		fmt.Printf("   URL: %s\n", videoUrl)
		fmt.Printf("   Duration: %s\n", video.Duration)
		fmt.Printf("   Views: %d\n", video.Views)
		fmt.Printf("   Author: %s\n", video.Author)
		fmt.Printf("   Description: %.200s...\n", video.Description)
		fmt.Println()

		// DynamoDBに保存する場合はVideoRecordを作成
		if l.saveToDB {
			description := video.Description
			if len(description) > 500 {
				description = description[:500] + "..."
			}
			
			item := types.PlaylistItem{
				Title:   video.Title,
				VideoId: video.ID,
			}
			
			videoRecord := dynamo.ConvertPlaylistItemToVideoRecord(
				item, 
				video.Duration.String(), 
				int64(video.Views), 
				video.Author, 
				description,
			)
			videoRecords = append(videoRecords, videoRecord)
		}
	}

	// DynamoDBに一括保存（改良版のリトライ機能付き）
	if l.saveToDB && len(videoRecords) > 0 {
		fmt.Printf("Saving %d videos to DynamoDB with retry logic...\n", len(videoRecords))
		if err := l.dynamoClient.SaveVideosWithRetry(ctx, videoRecords); err != nil {
			log.Printf("Failed to save videos to DynamoDB: %v", err)
			// フォールバックとして標準版を試行
			fmt.Printf("Attempting fallback save without advanced retry...\n")
			if fallbackErr := l.dynamoClient.SaveVideos(ctx, videoRecords); fallbackErr != nil {
				log.Printf("Fallback save also failed: %v", fallbackErr)
			} else {
				fmt.Printf("Fallback save successful!\n")
			}
		} else {
			fmt.Printf("Successfully saved all videos to DynamoDB!\n")
		}
	}

	return nil
}
