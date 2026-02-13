package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type DeviceData struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RowNum    int                `bson:"row_num" json:"row_num"`
	MQTT      string             `bson:"mqtt,omitempty" json:"mqtt,omitempty"`
	Inventory string             `bson:"inventory" json:"inventory"`
	UnitGUID  string             `bson:"unit_guid" json:"unit_guid"`
	MsgID     string             `bson:"msg_id" json:"msg_id"`
	Text      string             `bson:"text" json:"text"`
	Context   string             `bson:"context" json:"context"`
	Class     string             `bson:"class" json:"class"`
	Level     int                `bson:"level" json:"level"`
	Area      string             `bson:"area" json:"area"`
	Addr      string             `bson:"addr" json:"addr"`
	Block     string             `bson:"block" json:"block"`
	Type      string             `bson:"type" json:"type"`
	Bit       int                `bson:"bit" json:"bit"`
	InvertBit int                `bson:"invert_bit" json:"invert_bit"`
	FileName  string             `bson:"file_name" json:"file_name"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type ProcessedFile struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FileName    string             `bson:"file_name" json:"file_name"`
	FilePath    string             `bson:"file_path" json:"file_path"`
	ProcessedAt time.Time          `bson:"processed_at" json:"processed_at"`
	Status      string             `bson:"status" json:"status"`
	ErrorMsg    string             `bson:"error_msg,omitempty" json:"error_msg,omitempty"`
}

type ProcessingError struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FileName  string             `bson:"file_name" json:"file_name"`
	UnitGUID  string             `bson:"unit_guid,omitempty" json:"unit_guid,omitempty"`
	ErrorMsg  string             `bson:"error_msg" json:"error_msg"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}

type PaginatedResponse struct {
	Data       []DeviceData `json:"data"`
	Total      int64        `json:"total"`
	Page       int64        `json:"page"`
	Limit      int64        `json:"limit"`
	TotalPages int64        `json:"total_pages"`
}
