package lister

import (
	"fmt"
	"log"

	"youtube/types"

	"github.com/kkdai/youtube/v2"
)

// VideoLister handles YouTube video list operations
type VideoLister struct {
	client youtube.Client
}

// NewVideoLister creates a new VideoLister instance
func NewVideoLister() *VideoLister {
	return &VideoLister{
		client: youtube.Client{},
	}
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
		} else {
			fmt.Printf("   Duration: %s\n", video.Duration)
			fmt.Printf("   Views: %d\n", video.Views)
			fmt.Printf("   Author: %s\n", video.Author)
		}
		fmt.Println()
	}
	return nil
}

// ListVideosByIds lists videos by their IDs
func (l *VideoLister) ListVideosByIds(videoIds []string) error {
	fmt.Printf("Listing %d videos:\n\n", len(videoIds))

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
	}
	return nil
}
