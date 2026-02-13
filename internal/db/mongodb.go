package db

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/tsv-processor/internal/config"
	"github.com/tsv-processor/internal/models"
)

type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

type CollectionNames struct {
	DeviceData     string
	ProcessedFiles string
	ProcessingErrs string
}

var Collections = CollectionNames{
	DeviceData:     "device_data",
	ProcessedFiles: "processed_files",
	ProcessingErrs: "processing_errors",
}

func NewMongoDB(cfg *config.DatabaseConfig) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.GetMongoURI())

	if cfg.Username != "" && cfg.Password != "" {
		authDB := cfg.AuthDB
		if authDB == "" {
			authDB = "admin"
		}
		clientOpts.SetAuth(options.Credential{
			Username:   cfg.Username,
			Password:   cfg.Password,
			AuthSource: authDB,
		})
	}

	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	db := client.Database(cfg.Database)

	if err := createIndexes(ctx, db); err != nil {
		return nil, fmt.Errorf("failed to create indexes: %w", err)
	}

	return &MongoDB{
		client:   client,
		database: db,
	}, nil
}

func createIndexes(ctx context.Context, db *mongo.Database) error {
	deviceDataColl := db.Collection(Collections.DeviceData)
	deviceDataIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "unit_guid", Value: 1}},
			Options: options.Index().SetBackground(true),
		},
		{
			Keys:    bson.D{{Key: "file_name", Value: 1}},
			Options: options.Index().SetBackground(true),
		},
		{
			Keys:    bson.D{{Key: "timestamp", Value: -1}},
			Options: options.Index().SetBackground(true),
		},
		{
			Keys: bson.D{
				{Key: "unit_guid", Value: 1},
				{Key: "timestamp", Value: -1},
			},
			Options: options.Index().SetBackground(true),
		},
	}
	if _, err := deviceDataColl.Indexes().CreateMany(ctx, deviceDataIndexes); err != nil {
		return err
	}

	processedFilesColl := db.Collection(Collections.ProcessedFiles)
	processedFilesIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "file_name", Value: 1}},
			Options: options.Index().SetUnique(true).SetBackground(true),
		},
		{
			Keys:    bson.D{{Key: "processed_at", Value: -1}},
			Options: options.Index().SetBackground(true),
		},
	}
	if _, err := processedFilesColl.Indexes().CreateMany(ctx, processedFilesIndexes); err != nil {
		return err
	}

	processingErrsColl := db.Collection(Collections.ProcessingErrs)
	processingErrsIndexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "file_name", Value: 1}},
			Options: options.Index().SetBackground(true),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: -1}},
			Options: options.Index().SetBackground(true),
		},
	}
	_, err := processingErrsColl.Indexes().CreateMany(ctx, processingErrsIndexes)
	return err
}

func (db *MongoDB) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return db.client.Disconnect(ctx)
}

func (db *MongoDB) IsFileProcessed(ctx context.Context, fileName string) (bool, error) {
	collection := db.database.Collection(Collections.ProcessedFiles)

	filter := bson.M{"file_name": fileName}
	count, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

func (db *MongoDB) SaveProcessedFile(ctx context.Context, file *models.ProcessedFile) error {
	collection := db.database.Collection(Collections.ProcessedFiles)

	if file.ID.IsZero() {
		file.ID = primitive.NewObjectID()
	}

	_, err := collection.InsertOne(ctx, file)
	return err
}

func (db *MongoDB) SaveDeviceData(ctx context.Context, data []*models.DeviceData) error {
	if len(data) == 0 {
		return nil
	}

	collection := db.database.Collection(Collections.DeviceData)

	documents := make([]interface{}, len(data))
	for i, d := range data {
		if d.ID.IsZero() {
			d.ID = primitive.NewObjectID()
		}
		documents[i] = d
	}

	_, err := collection.InsertMany(ctx, documents)
	return err
}

func (db *MongoDB) SaveProcessingError(ctx context.Context, err *models.ProcessingError) error {
	collection := db.database.Collection(Collections.ProcessingErrs)

	if err.ID.IsZero() {
		err.ID = primitive.NewObjectID()
	}

	_, errDb := collection.InsertOne(ctx, err)
	return errDb
}

func (db *MongoDB) GetDeviceDataByUnitGUID(ctx context.Context, unitGUID string, page, limit int64) (*models.PaginatedResponse, error) {
	collection := db.database.Collection(Collections.DeviceData)

	filter := bson.M{"unit_guid": unitGUID}

	total, err := collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	skip := (page - 1) * limit

	findOptions := options.Find().
		SetSort(bson.D{{Key: "timestamp", Value: -1}}).
		SetSkip(skip).
		SetLimit(limit)

	cursor, err := collection.Find(ctx, filter, findOptions)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var data []models.DeviceData
	if err := cursor.All(ctx, &data); err != nil {
		return nil, err
	}

	totalPages := total / limit
	if total%limit > 0 {
		totalPages++
	}

	return &models.PaginatedResponse{
		Data:       data,
		Total:      total,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
	}, nil
}

func (db *MongoDB) GetDeviceDataByFile(ctx context.Context, fileName string) ([]models.DeviceData, error) {
	collection := db.database.Collection(Collections.DeviceData)

	filter := bson.M{"file_name": fileName}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var data []models.DeviceData
	if err := cursor.All(ctx, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func (db *MongoDB) GetProcessingErrors(ctx context.Context, fileName string) ([]models.ProcessingError, error) {
	collection := db.database.Collection(Collections.ProcessingErrs)

	filter := bson.M{"file_name": fileName}
	cursor, err := collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var errors []models.ProcessingError
	if err := cursor.All(ctx, &errors); err != nil {
		return nil, err
	}

	return errors, nil
}
