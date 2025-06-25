package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/kkdai/youtube/v2"
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

    // 1. アップロード動画プレイリストIDを取得
    url := fmt.Sprintf("https://www.googleapis.com/youtube/v3/channels?part=contentDetails&id=%s&key=%s", channelId, apiKey)
    resp, err := http.Get(url)
    if err != nil {
        log.Fatalf("Failed to get channel details: %v", err)
    }
    defer resp.Body.Close()
    body, err := io.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("Failed to read response body: %v", err)
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
    json.Unmarshal(body, &channelResp)
    uploadsPlaylistId := channelResp.Items[0].ContentDetails.RelatedPlaylists.Uploads

    // 2. プレイリスト内の動画一覧を取得
    url = fmt.Sprintf("https://www.googleapis.com/youtube/v3/playlistItems?part=snippet&playlistId=%s&maxResults=50&key=%s", uploadsPlaylistId, apiKey)
    resp, err = http.Get(url)
    if err != nil {
        log.Fatalf("Failed to get playlist items: %v", err)
    }
    defer resp.Body.Close()
    body, err = io.ReadAll(resp.Body)
    if err != nil {
        log.Fatalf("Failed to read playlist response body: %v", err)
    }

    var playlistResp struct {
        Items []struct {
            Snippet struct {
                Title    string `json:"title"`
                ResourceId struct {
                    VideoId string `json:"videoId"`
                } `json:"resourceId"`
            } `json:"snippet"`
        } `json:"items"`
    }
    json.Unmarshal(body, &playlistResp)

		client := youtube.Client{}
    for _, item := range playlistResp.Items {
        fmt.Printf("Title: %s, VideoId: %s\n", item.Snippet.Title, item.Snippet.ResourceId.VideoId)
				// download動画のURLを生成
				videoUrl := fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.Snippet.ResourceId.VideoId)
				fmt.Println("Download URL:", videoUrl)
				video,err := client.GetVideo(videoUrl)
				if err != nil{
					log.Fatalf("Failed to get video: %v", err)
				}

				// 音声付きの適切な形式を取得
				formats := video.Formats.WithAudioChannels()
				if len(formats) == 0 {
						log.Fatal("No formats with audio found")
				}

				// または品質で絞り込む場合
				var format *youtube.Format
				for _, f := range formats {
					if f.MimeType == "video/mp4" && 
						 (f.QualityLabel == "720p" || f.QualityLabel == "480p") {
							format = &f
							break
					}else{
						fmt.Printf("Skipping format: %s, Quality: %s", f.MimeType, f.QualityLabel)
					}
				}

				if format == nil {
						format = &formats[0]
				}

				// 音声しか取れない。
				stream, _, err := client.GetStream(video, format)
				if err != nil {
						log.Fatal(err)
				}
				defer stream.Close()
				
				file, err := os.Create(item.Snippet.ResourceId.VideoId + ".mp4")
				if err != nil {
						log.Fatal(err)
				}
				defer file.Close()
		
				_, err = io.Copy(file, stream)
				if err != nil {
						log.Fatal(err)
				}

				fmt.Println("Downloaded:", item.Snippet.Title)
    }
}
