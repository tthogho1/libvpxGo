# YouTube Video Manager

このプロジェクトは YouTube 動画の一覧取得とダウンロードを行う Go アプリケーションです。

## 機能

- **list**: プレイリストまたは指定した動画の一覧表示
- **formats**: 動画の利用可能なフォーマット情報表示
- **download**: 動画のダウンロード
- **DynamoDB 連携**: 動画情報を AWS DynamoDB に自動保存（オプション）
  - 自動リトライ機能付きバッチ書き込み
  - UnprocessedItems の適切な処理
  - 指数バックオフによるスロットリング対策
- **Transcribe 管理**: 動画の transcribe 処理状況を管理

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

### 6. DynamoDB への保存付きリスト表示

```bash
# プレイリストの一覧をDynamoDBに保存
go run main.go -mode=list -save-db

# カスタムテーブル名で保存
go run main.go -mode=list -save-db -table=my_youtube_videos

# 特定の動画IDをDynamoDBに保存
go run main.go -mode=list -ids="VIDEO_ID1,VIDEO_ID2" -save-db
```

### 7. Transcribe 管理機能

```bash
# Transcribe未実施の動画一覧を表示
go run main.go -mode=list-untranscribed -table=youtube_videos

# Transcribe実施済みの動画一覧を表示
go run main.go -mode=list-transcribed -table=youtube_videos

# 特定の動画のTranscribeステータスを完了に更新
go run main.go -mode=transcribe-status -ids="VIDEO_ID1,VIDEO_ID2" -transcribed=true -table=youtube_videos

# 特定の動画のTranscribeステータスを未実施に更新
go run main.go -mode=transcribe-status -ids="VIDEO_ID1,VIDEO_ID2" -transcribed=false -table=youtube_videos
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

**注意**: `.env`ファイルには機密情報が含まれるため、Git にコミットしないでください。このファイルは`.gitignore`で除外されています。

### AWS DynamoDB 設定（オプション）

DynamoDB 連携機能を使用する場合は、AWS 認証情報の設定が必要です：

#### 方法 1: AWS CLI

```bash
aws configure
```

#### 方法 2: 環境変数

```env
AWS_ACCESS_KEY_ID=your_access_key
AWS_SECRET_ACCESS_KEY=your_secret_key
AWS_REGION=us-east-1
```

#### 方法 3: IAM ロール（EC2 インスタンス使用時）

EC2 インスタンスでアプリケーションを実行する場合、IAM ロールを使用して認証できます。

#### DynamoDB テーブル

- アプリケーションが自動的にテーブルを作成します
- デフォルトテーブル名: `youtube_videos`
- `-table`フラグでカスタム名を指定可能
- 課金モード: Pay-per-request

**バッチ書き込み機能:**

- 25 件ずつのバッチ処理で効率的に保存
- UnprocessedItems の自動リトライ（最大 5 回）
- スロットリング対策の指数バックオフ
- 部分的な失敗にも対応したフォールバック機能

**テーブル構造:**

```json
{
  "video_id": "string", // パーティションキー
  "title": "string",
  "author": "string",
  "duration": "string",
  "views": "number",
  "description": "string",
  "url": "string",
  "transcribed": "boolean", // Transcribe処理完了フラグ
  "created_at": "timestamp",
  "updated_at": "timestamp"
}
```

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
├── lister/
│   └── lister.go        # 動画一覧表示機能
└── dynamo/
    └── dynamo.go        # DynamoDB操作機能
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

### Transcribe 管理

```bash
go run main.go -mode=list-untranscribed -table=youtube_videos
```

出力例：

```
Found 3 untranscribed videos:

1. Title: サンプル動画1
   Video ID: abc123
   URL: https://www.youtube.com/watch?v=abc123
   Author: チャンネル名
   Duration: 5m30s
   Views: 1000
   Transcribed: false

2. Title: サンプル動画2
   ...
```

**Transcribe ステータス更新:**

```bash
go run main.go -mode=transcribe-status -ids="abc123" -transcribed=true -table=youtube_videos
```

出力例：

```
Updated transcribe status for video abc123 to true
```
