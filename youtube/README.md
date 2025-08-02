# YouTube Video Manager

このプロジェクトは YouTube 動画の一覧取得とダウンロードを行う Go アプリケーションです。

## 機能

- **list**: プレイリストまたは指定した動画の一覧表示
- **formats**: 動画の利用可能なフォーマット情報表示
- **download**: 動画のダウンロード

## 使用方法

### 1. プレイリストの一覧表示

```bash
go run main.go -mode=list
```

### 2. 特定の動画 ID の一覧表示

```bash
go run main.go -mode=list -ids="VIDEO_ID1,VIDEO_ID2"
```

### 3. 特定の URL の一覧表示

```bash
go run main.go -mode=list -urls="https://www.youtube.com/watch?v=VIDEO_ID1,https://www.youtube.com/watch?v=VIDEO_ID2"
```

### 4. 動画のフォーマット情報表示

```bash
go run main.go -mode=formats -urls="https://www.youtube.com/watch?v=VIDEO_ID"
```

### 5. 動画のダウンロード（従来の機能）

```bash
go run main.go -mode=download
```

## 設定

### 環境変数の設定

1. `.env.example`ファイルを`.env`にコピーしてください：
   ```bash
   cp .env.example .env
   ```

2. `.env`ファイルを編集して、実際の値を設定してください：

```env
# YouTube Data API v3 API Key
# Google Cloud Consoleから取得: https://console.developers.google.com/
YOUTUBE_API_KEY=your_actual_youtube_api_key

# あなたのYouTubeチャンネルID
# YouTube Studioの設定 -> チャンネル -> 詳細設定から確認できます
YOUR_CHANNEL_ID=your_actual_channel_id

# ダウンロードディレクトリ（オプション、デフォルト: ./downloads）
DOWNLOAD_DIR=./downloads
```

**注意**: `.env`ファイルには機密情報が含まれるため、Gitにコミットしないでください。このファイルは`.gitignore`で除外されています。

## プロジェクト構造

```
youtube/
├── main.go              # メインアプリケーション
├── types/
│   └── types.go         # 共通の型定義
├── api/
│   └── youtube_api.go   # YouTube API クライアント
├── downloader/
│   └── downloader.go    # 動画ダウンロード機能
└── lister/
    └── lister.go        # 動画一覧表示機能
```

## 例

### プレイリストの一覧表示

```bash
go run main.go -mode=list
```

出力例：

```
Found 10 videos in playlist:

1. Title: サンプル動画1
   Video ID: abc123
   URL: https://www.youtube.com/watch?v=abc123
   Duration: 5m30s
   Views: 1000
   Author: チャンネル名

2. Title: サンプル動画2
   ...
```

### フォーマット情報表示

```bash
go run main.go -mode=formats -urls="https://www.youtube.com/watch?v=abc123"
```

出力例：

```
Video Title: サンプル動画
Video ID: abc123
Duration: 5m30s
Author: チャンネル名
View Count: 1000

--- Available Formats ---
Format 1:
  MIME Type: video/mp4
  Quality: 720p
  Bitrate: 2000000
  Audio Channels: 2
  Video Resolution: 1280x720
  FPS: 30
  ---
```
