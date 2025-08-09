package dynamo

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"youtube/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	dynamoTypes "github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

// VideoRecord represents a video record in DynamoDB
type VideoRecord struct {
	VideoID     string    `dynamodbav:"video_id" json:"video_id"`
	Title       string    `dynamodbav:"title" json:"title"`
	Author      string    `dynamodbav:"author" json:"author"`
	Duration    string    `dynamodbav:"duration" json:"duration"`
	Views       int64     `dynamodbav:"views" json:"views"`
	Description string    `dynamodbav:"description" json:"description"`
	URL         string    `dynamodbav:"url" json:"url"`
	Transcribed bool      `dynamodbav:"transcribed" json:"transcribed"`
	CreatedAt   time.Time `dynamodbav:"created_at" json:"created_at"`
	UpdatedAt   time.Time `dynamodbav:"updated_at" json:"updated_at"`
}

// DynamoDBClient handles DynamoDB operations
type DynamoDBClient struct {
	client    *dynamodb.Client
	tableName string
}

// NewDynamoDBClient creates a new DynamoDB client
func NewDynamoDBClient(tableName string) (*DynamoDBClient, error) {
	// 環境変数からAWS認証情報を取得
	accessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	secretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	region := os.Getenv("AWS_REGION")

	// 認証情報の確認
	if accessKey == "" {
		return nil, fmt.Errorf("AWS_ACCESS_KEY_ID environment variable is not set")
	}
	if secretKey == "" {
		return nil, fmt.Errorf("AWS_SECRET_ACCESS_KEY environment variable is not set")
	}
	if region == "" {
		return nil, fmt.Errorf("AWS_REGION environment variable is not set")
	}

	log.Printf("Using AWS Region: %s", region)
	log.Printf("Using AWS Access Key: %s...", accessKey[:min(10, len(accessKey))])

	// AWS設定を作成（環境変数の認証情報を明示的に指定）
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion(region),
		config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
			accessKey,
			secretKey,
			"", // session token (通常は空)
		)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %v", err)
	}

	client := dynamodb.NewFromConfig(cfg)

	return &DynamoDBClient{
		client:    client,
		tableName: tableName,
	}, nil
}

// min helper function
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// CreateTableIfNotExists creates the DynamoDB table if it doesn't exist
func (d *DynamoDBClient) CreateTableIfNotExists(ctx context.Context) error {
	// テーブルが存在するかチェック
	_, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(d.tableName),
	})

	if err == nil {
		// テーブルが既に存在する
		log.Printf("Table %s already exists", d.tableName)
		return nil
	}

	// テーブルが存在しない場合は作成
	log.Printf("Creating table %s...", d.tableName)

	input := &dynamodb.CreateTableInput{
		TableName: aws.String(d.tableName),
		KeySchema: []dynamoTypes.KeySchemaElement{
			{
				AttributeName: aws.String("video_id"),
				KeyType:       dynamoTypes.KeyTypeHash,
			},
		},
		AttributeDefinitions: []dynamoTypes.AttributeDefinition{
			{
				AttributeName: aws.String("video_id"),
				AttributeType: dynamoTypes.ScalarAttributeTypeS,
			},
		},
		BillingMode: dynamoTypes.BillingModePayPerRequest,
	}

	_, err = d.client.CreateTable(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to create table: %v", err)
	}

	// テーブルがアクティブになるまで待機
	waiter := dynamodb.NewTableExistsWaiter(d.client)
	err = waiter.Wait(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(d.tableName),
	}, 5*time.Minute)

	if err != nil {
		return fmt.Errorf("failed to wait for table creation: %v", err)
	}

	log.Printf("Table %s created successfully", d.tableName)
	return nil
}

// SaveVideo saves a single video record to DynamoDB
func (d *DynamoDBClient) SaveVideo(ctx context.Context, video VideoRecord) error {
	// タイムスタンプを設定
	now := time.Now()
	video.CreatedAt = now
	video.UpdatedAt = now

	// 構造体をDynamoDB属性値に変換
	item, err := attributevalue.MarshalMap(video)
	if err != nil {
		return fmt.Errorf("failed to marshal video record: %v", err)
	}

	// DynamoDBに保存
	_, err = d.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(d.tableName),
		Item:      item,
	})

	if err != nil {
		return fmt.Errorf("failed to save video to DynamoDB: %v", err)
	}

	log.Printf("Saved video to DynamoDB: %s - %s", video.VideoID, video.Title)
	return nil
}

// SaveVideos saves multiple video records to DynamoDB
func (d *DynamoDBClient) SaveVideos(ctx context.Context, videos []VideoRecord) error {
	if len(videos) == 0 {
		return nil
	}

	// バッチ書き込みのためのアイテムを準備
	writeRequests := make([]dynamoTypes.WriteRequest, 0, len(videos))

	for _, video := range videos {
		// タイムスタンプを設定
		now := time.Now()
		video.CreatedAt = now
		video.UpdatedAt = now

		// 構造体をDynamoDB属性値に変換
		item, err := attributevalue.MarshalMap(video)
		if err != nil {
			log.Printf("Failed to marshal video record %s: %v", video.VideoID, err)
			continue
		}

		writeRequests = append(writeRequests, dynamoTypes.WriteRequest{
			PutRequest: &dynamoTypes.PutRequest{
				Item: item,
			},
		})
	}

	// DynamoDBのバッチ書き込み制限（25アイテム）に対応
	batchSize := 25
	for i := 0; i < len(writeRequests); i += batchSize {
		end := i + batchSize
		if end > len(writeRequests) {
			end = len(writeRequests)
		}

		batch := writeRequests[i:end]
		
		// UnprocessedItemsを処理するためのリトライロジック
		remainingItems := map[string][]dynamoTypes.WriteRequest{
			d.tableName: batch,
		}
		
		maxRetries := 3
		retryCount := 0
		
		for len(remainingItems[d.tableName]) > 0 && retryCount < maxRetries {
			result, err := d.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
				RequestItems: remainingItems,
			})

			if err != nil {
				return fmt.Errorf("failed to batch write items to DynamoDB (retry %d): %v", retryCount, err)
			}

			// UnprocessedItemsがある場合は次のリトライで処理
			if result.UnprocessedItems != nil && len(result.UnprocessedItems[d.tableName]) > 0 {
				remainingItems = result.UnprocessedItems
				retryCount++
				log.Printf("Batch write partially successful. Retrying %d unprocessed items (attempt %d/%d)", 
					len(remainingItems[d.tableName]), retryCount+1, maxRetries)
				
				// 指数バックオフによる待機
				if retryCount < maxRetries {
					waitTime := time.Duration(retryCount) * time.Second
					log.Printf("Waiting %v before retry...", waitTime)
					time.Sleep(waitTime)
				}
			} else {
				// すべてのアイテムが正常に処理された
				remainingItems[d.tableName] = nil
				break
			}
		}
		
		// 最大リトライ回数に達してもUnprocessedItemsが残っている場合
		if len(remainingItems[d.tableName]) > 0 {
			log.Printf("Warning: %d items could not be written after %d retries", 
				len(remainingItems[d.tableName]), maxRetries)
			// エラーを返すか、ログ出力のみにするかは要件に応じて調整
		}

		log.Printf("Saved batch of %d videos to DynamoDB", len(batch))
	}

	log.Printf("Successfully saved %d videos to DynamoDB", len(videos))
	return nil
}

// SaveVideosWithRetry saves multiple video records to DynamoDB with advanced retry logic
func (d *DynamoDBClient) SaveVideosWithRetry(ctx context.Context, videos []VideoRecord) error {
	if len(videos) == 0 {
		return nil
	}

	// バッチ書き込みのためのアイテムを準備
	writeRequests := make([]dynamoTypes.WriteRequest, 0, len(videos))

	for _, video := range videos {
		// タイムスタンプを設定
		now := time.Now()
		video.CreatedAt = now
		video.UpdatedAt = now

		// 構造体をDynamoDB属性値に変換
		item, err := attributevalue.MarshalMap(video)
		if err != nil {
			log.Printf("Failed to marshal video record %s: %v", video.VideoID, err)
			continue
		}

		writeRequests = append(writeRequests, dynamoTypes.WriteRequest{
			PutRequest: &dynamoTypes.PutRequest{
				Item: item,
			},
		})
	}

	totalProcessed := 0
	totalFailed := 0
	batchSize := 25

	for i := 0; i < len(writeRequests); i += batchSize {
		end := i + batchSize
		if end > len(writeRequests) {
			end = len(writeRequests)
		}

		batch := writeRequests[i:end]
		processed, failed := d.writeBatchWithRetry(ctx, batch)
		totalProcessed += processed
		totalFailed += failed
	}

	log.Printf("Batch write completed: %d successful, %d failed out of %d total videos", 
		totalProcessed, totalFailed, len(videos))

	if totalFailed > 0 {
		return fmt.Errorf("failed to write %d out of %d items to DynamoDB", totalFailed, len(videos))
	}

	return nil
}

// writeBatchWithRetry handles a single batch with retry logic
func (d *DynamoDBClient) writeBatchWithRetry(ctx context.Context, batch []dynamoTypes.WriteRequest) (int, int) {
	remainingItems := map[string][]dynamoTypes.WriteRequest{
		d.tableName: batch,
	}
	
	originalCount := len(batch)
	maxRetries := 5
	retryCount := 0
	
	for len(remainingItems[d.tableName]) > 0 && retryCount < maxRetries {
		result, err := d.client.BatchWriteItem(ctx, &dynamodb.BatchWriteItemInput{
			RequestItems: remainingItems,
		})

		if err != nil {
			log.Printf("BatchWriteItem error (retry %d): %v", retryCount, err)
			retryCount++
			
			// エラーの場合は指数バックオフで待機
			if retryCount < maxRetries {
				waitTime := time.Duration(1<<retryCount) * time.Second // 2^retryCount秒
				log.Printf("Waiting %v before retry due to error...", waitTime)
				time.Sleep(waitTime)
			}
			continue
		}

		// UnprocessedItemsの処理
		if result.UnprocessedItems != nil && len(result.UnprocessedItems[d.tableName]) > 0 {
			remainingItems = result.UnprocessedItems
			retryCount++
			log.Printf("Batch partially successful. %d unprocessed items remaining (attempt %d/%d)", 
				len(remainingItems[d.tableName]), retryCount+1, maxRetries)
			
			// スロットリング対策の待機（線形バックオフ）
			if retryCount < maxRetries {
				waitTime := time.Duration(retryCount*500) * time.Millisecond
				log.Printf("Waiting %v before retry for unprocessed items...", waitTime)
				time.Sleep(waitTime)
			}
		} else {
			// すべてのアイテムが正常に処理された
			remainingItems[d.tableName] = nil
			break
		}
	}
	
	processed := originalCount - len(remainingItems[d.tableName])
	failed := len(remainingItems[d.tableName])
	
	if failed > 0 {
		log.Printf("Warning: %d items could not be written after %d retries", failed, maxRetries)
	} else {
		log.Printf("Successfully saved batch of %d videos to DynamoDB", originalCount)
	}
	
	return processed, failed
}

// GetVideo retrieves a video record from DynamoDB by video ID
func (d *DynamoDBClient) GetVideo(ctx context.Context, videoID string) (*VideoRecord, error) {
	result, err := d.client.GetItem(ctx, &dynamodb.GetItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]dynamoTypes.AttributeValue{
			"video_id": &dynamoTypes.AttributeValueMemberS{Value: videoID},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get video from DynamoDB: %v", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("video not found: %s", videoID)
	}

	var video VideoRecord
	err = attributevalue.UnmarshalMap(result.Item, &video)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal video record: %v", err)
	}

	return &video, nil
}

// UpdateTranscribeStatus updates the transcribe status of a video
func (d *DynamoDBClient) UpdateTranscribeStatus(ctx context.Context, videoID string, transcribed bool) error {
	now := time.Now()
	
	_, err := d.client.UpdateItem(ctx, &dynamodb.UpdateItemInput{
		TableName: aws.String(d.tableName),
		Key: map[string]dynamoTypes.AttributeValue{
			"video_id": &dynamoTypes.AttributeValueMemberS{Value: videoID},
		},
		UpdateExpression: aws.String("SET transcribed = :transcribed, updated_at = :updated_at"),
		ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
			":transcribed": &dynamoTypes.AttributeValueMemberBOOL{Value: transcribed},
			":updated_at":  &dynamoTypes.AttributeValueMemberS{Value: now.Format(time.RFC3339)},
		},
	})

	if err != nil {
		return fmt.Errorf("failed to update transcribe status for video %s: %v", videoID, err)
	}

	log.Printf("Updated transcribe status for video %s to %t", videoID, transcribed)
	return nil
}

// GetUntranscribedVideos retrieves all videos that haven't been transcribed yet
func (d *DynamoDBClient) GetUntranscribedVideos(ctx context.Context) ([]VideoRecord, error) {
	var videos []VideoRecord
	var lastEvaluatedKey map[string]dynamoTypes.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:        aws.String(d.tableName),
			FilterExpression: aws.String("transcribed = :transcribed"),
			ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
				":transcribed": &dynamoTypes.AttributeValueMemberBOOL{Value: false},
			},
		}

		if lastEvaluatedKey != nil {
			input.ExclusiveStartKey = lastEvaluatedKey
		}

		result, err := d.client.Scan(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to scan untranscribed videos: %v", err)
		}

		// 結果をVideoRecordに変換
		for _, item := range result.Items {
			var video VideoRecord
			err = attributevalue.UnmarshalMap(item, &video)
			if err != nil {
				log.Printf("Failed to unmarshal video record: %v", err)
				continue
			}
			videos = append(videos, video)
		}

		// ページネーションの処理
		if result.LastEvaluatedKey == nil {
			break
		}
		lastEvaluatedKey = result.LastEvaluatedKey
	}

	log.Printf("Found %d untranscribed videos", len(videos))
	return videos, nil
}

// GetTranscribedVideos retrieves all videos that have been transcribed
func (d *DynamoDBClient) GetTranscribedVideos(ctx context.Context) ([]VideoRecord, error) {
	var videos []VideoRecord
	var lastEvaluatedKey map[string]dynamoTypes.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:        aws.String(d.tableName),
			FilterExpression: aws.String("transcribed = :transcribed"),
			ExpressionAttributeValues: map[string]dynamoTypes.AttributeValue{
				":transcribed": &dynamoTypes.AttributeValueMemberBOOL{Value: true},
			},
		}

		if lastEvaluatedKey != nil {
			input.ExclusiveStartKey = lastEvaluatedKey
		}

		result, err := d.client.Scan(ctx, input)
		if err != nil {
			return nil, fmt.Errorf("failed to scan transcribed videos: %v", err)
		}

		// 結果をVideoRecordに変換
		for _, item := range result.Items {
			var video VideoRecord
			err = attributevalue.UnmarshalMap(item, &video)
			if err != nil {
				log.Printf("Failed to unmarshal video record: %v", err)
				continue
			}
			videos = append(videos, video)
		}

		// ページネーションの処理
		if result.LastEvaluatedKey == nil {
			break
		}
		lastEvaluatedKey = result.LastEvaluatedKey
	}

	log.Printf("Found %d transcribed videos", len(videos))
	return videos, nil
}

// ConvertPlaylistItemToVideoRecord converts a PlaylistItem to VideoRecord
func ConvertPlaylistItemToVideoRecord(item types.PlaylistItem, duration string, views int64, author, description string) VideoRecord {
	return VideoRecord{
		VideoID:     item.VideoId,
		Title:       item.Title,
		Author:      author,
		Duration:    duration,
		Views:       views,
		Description: description,
		URL:         fmt.Sprintf("https://www.youtube.com/watch?v=%s", item.VideoId),
		Transcribed: false, // デフォルトではtranscribe未実施
	}
}

// TestConnection tests the DynamoDB connection and basic operations
func (d *DynamoDBClient) TestConnection(ctx context.Context) error {
	log.Printf("Testing DynamoDB connection...")

	// テーブルの存在確認
	result, err := d.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: aws.String(d.tableName),
	})

	if err != nil {
		log.Printf("Failed to describe table %s: %v", d.tableName, err)
		return fmt.Errorf("failed to describe table: %v", err)
	}

	log.Printf("✓ Table %s exists", d.tableName)
	log.Printf("✓ Table status: %s", result.Table.TableStatus)
	log.Printf("✓ Item count: %d", *result.Table.ItemCount)

	// テストアイテムを書き込んでみる
	testVideo := VideoRecord{
		VideoID:     fmt.Sprintf("test_%d", time.Now().Unix()),
		Title:       "Test Video",
		Author:      "Test Author", 
		Duration:    "00:01:00",
		Views:       0,
		Description: "Test description for connection test",
		URL:         "https://www.youtube.com/watch?v=test",
		Transcribed: false,
	}

	log.Printf("Attempting to save test video: %s", testVideo.VideoID)

	err = d.SaveVideo(ctx, testVideo)
	if err != nil {
		log.Printf("Failed to save test video: %v", err)
		return fmt.Errorf("failed to save test video: %v", err)
	}

	log.Printf("✓ Test video saved successfully")

	// テストアイテムを読み込んでみる
	retrievedVideo, err := d.GetVideo(ctx, testVideo.VideoID)
	if err != nil {
		log.Printf("Failed to retrieve test video: %v", err)
		return fmt.Errorf("failed to retrieve test video: %v", err)
	}

	log.Printf("✓ Test video retrieved successfully: %s", retrievedVideo.Title)
	log.Printf("✓ DynamoDB connection test completed successfully")
	return nil
}
