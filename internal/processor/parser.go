package processor

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tsv-processor/internal/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type TSVParser struct{}

func NewTSVParser() *TSVParser {
	return &TSVParser{}
}

func (p *TSVParser) Parse(filePath, fileName string) ([]*models.DeviceData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = '\t'
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to skip first header: %w", err)
	}

	_, err = reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to skip second header: %w", err)
	}

	var records []*models.DeviceData
	rowNum := 3

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("error reading row %d: %w", rowNum, err)
		}

		var cleanRow []string
		for _, field := range row {
			trimmed := strings.TrimSpace(field)
			if trimmed != "" {
				cleanRow = append(cleanRow, trimmed)
			}
		}
		row = cleanRow

		record := &models.DeviceData{
			ID:        primitive.NewObjectID(),
			FileName:  fileName,
			CreatedAt: time.Now(),

			RowNum:    mustAtoi(row[0]),
			Inventory: row[1],
			UnitGUID:  row[2],
			MsgID:     row[3],
			Text:      row[4],
			Class:     row[5],
			Level:     mustAtoi(row[6]),
			Area:      row[7],
			Addr:      row[8],
		}

		if record.UnitGUID == "" || record.MsgID == "" {
			return nil, fmt.Errorf("row %d: missing required fields", rowNum)
		}

		records = append(records, record)
		rowNum++
	}

	return records, nil
}

func mustAtoi(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}
