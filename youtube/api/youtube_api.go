package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"youtube/downloader"
)

// YouTubeAPI handles YouTube API interactions
type YouTubeAPI struct {
	apiKey    string
	channelId string
}

// NewYouTubeAPI creates a new YouTubeAPI instance
func NewYouTubeAPI(apiKey, channelId string) *YouTubeAPI {
	return &YouTubeAPI{
		apiKey:    apiKey,
		channelId: channelId,
	}
}

// GetPlaylistItems retrieves all video items from the channel's uploads playlist
func (api *YouTubeAPI) GetPlaylistItems() ([]downloader.PlaylistItem, error) {
	// 1. アップロード動画プレイリストIDを取得
	uploadsPlaylistId, err := api.getUploadsPlaylistId()
	if err != nil {
		return nil, fmt.Errorf("failed to get uploads playlist ID: %v", err)
	}

	// 2. プレイリスト内の動画一覧を取得
	items, err := api.getPlaylistVideos(uploadsPlaylistId)
	if err != nil {
		return nil, fmt.Errorf("failed to get playlist videos: %v", err)
	}

	return items, nil
}

// getUploadsPlaylistId retrieves the uploads playlist ID for the channel
func (api *YouTubeAPI) getUploadsPlaylistId() (string, error) {
	url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/channels?part=contentDetails&id=%s&key=%s", 
		api.channelId, api.apiKey)
	
	resp, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to get channel details: %v", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %v", err)
	}

	var channelResp struct {
		Items []struct {
			ContentDetails struct {
				RelatedPlaylists struct {
					Uploads string `json:"uploads"`
				} `json:"relatedPlaylists"`
			} `json:"contentDetails"`
		} `json:"items"`
	}
	
	if err := json.Unmarshal(body, &channelResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal channel response: %v", err)
	}
	
	if len(channelResp.Items) == 0 {
		return "", fmt.Errorf("no channel found")
	}
	
	return channelResp.Items[0].ContentDetails.RelatedPlaylists.Uploads, nil
}

// getPlaylistVideos retrieves all videos from the specified playlist
func (api *YouTubeAPI) getPlaylistVideos(playlistId string) ([]downloader.PlaylistItem, error) {
	var allItems []downloader.PlaylistItem
	nextPageToken := ""

	for {
		// URLを構築（nextPageTokenがある場合は追加）
		url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=%s&maxResults=50&key=%s", 
			playlistId, api.apiKey)
		
		if nextPageToken != "" {
			url = fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=%s&maxResults=50&pageToken=%s&key=%s",
				playlistId, nextPageToken, api.apiKey)
		}
		
		resp, err := http.Get(url)
		if err != nil {
			return nil, fmt.Errorf("failed to get playlist items: %v", err)
		}
		defer resp.Body.Close()
		
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read playlist response body: %v", err)
		}

		var playlistResp struct {
			Items []struct {
				Snippet struct {
					Title      string `json:"title"`
					ResourceId struct {
						VideoId string `json:"videoId"`
					} `json:"resourceId"`
				} `json:"snippet"`
			} `json:"items"`
			NextPageToken string `json:"nextPageToken"`
		}
		
		if err := json.Unmarshal(body, &playlistResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal playlist response: %v", err)
		}

		// 現在のページのアイテムを追加
		for _, item := range playlistResp.Items {
			allItems = append(allItems, downloader.PlaylistItem{
				Title:   item.Snippet.Title,
				VideoId: item.Snippet.ResourceId.VideoId,
			})
		}

		// nextPageTokenがない場合はループを終了
		if playlistResp.NextPageToken == "" {
			break
		}
		
		nextPageToken = playlistResp.NextPageToken
		fmt.Printf("取得済み: %d件, 次のページトークン: %s\n", len(allItems), nextPageToken)
	}

	fmt.Printf("合計取得件数: %d件\n", len(allItems))
	return allItems, nil
}
