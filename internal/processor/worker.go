package processor

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/tsv-processor/internal/config"
	"github.com/tsv-processor/internal/db"
	"github.com/tsv-processor/internal/generator"
	"github.com/tsv-processor/internal/models"
)

type Job struct {
	FilePath string
	FileName string
}

type WorkerPool struct {
	db        *db.MongoDB
	parser    *TSVParser
	generator *generator.ReportGenerator
	jobQueue  chan Job
	workers   int
	cfg       *config.WatcherConfig
}

func NewWorkerPool(db *db.MongoDB, cfg *config.WatcherConfig) *WorkerPool {
	return &WorkerPool{
		db:        db,
		parser:    NewTSVParser(),
		generator: generator.NewReportGenerator(cfg.OutputDir),
		jobQueue:  make(chan Job, 100),
		workers:   cfg.Workers,
		cfg:       cfg,
	}
}

func (wp *WorkerPool) Start(ctx context.Context) {
	for i := 0; i < wp.workers; i++ {
		go wp.worker(ctx, i)
	}

	go wp.watchDirectory(ctx)
}

func (wp *WorkerPool) watchDirectory(ctx context.Context) {
	ticker := time.NewTicker(wp.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			wp.scanDirectory()
		}
	}
}

func (wp *WorkerPool) scanDirectory() {
	files, err := filepath.Glob(filepath.Join(wp.cfg.InputDir, "*.tsv"))
	if err != nil {
		log.Printf("Error scanning directory: %v", err)
		return
	}

	for _, filePath := range files {
		fileName := filepath.Base(filePath)

		processed, err := wp.db.IsFileProcessed(context.Background(), fileName)
		if err != nil {
			log.Printf("Error checking processed file: %v", err)
			continue
		}

		if processed {
			log.Printf("File already processed: %s", fileName)
			continue
		}

		select {
		case wp.jobQueue <- Job{
			FilePath: filePath,
			FileName: fileName,
		}:
			log.Printf("Added job to queue: %s", fileName)
		default:
			log.Printf("Job queue is full, skipping: %s", fileName)
		}
	}
}

func (wp *WorkerPool) worker(ctx context.Context, id int) {
	log.Printf("Worker %d started", id)

	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping", id)
			return
		case job := <-wp.jobQueue:
			wp.processJob(ctx, job)
		}
	}
}

func (wp *WorkerPool) processJob(ctx context.Context, job Job) {
	log.Printf("Worker processing file: %s", job.FileName)

	processedFile := &models.ProcessedFile{
		ID:          primitive.NewObjectID(),
		FileName:    job.FileName,
		FilePath:    job.FilePath,
		ProcessedAt: time.Now(),
		Status:      "success",
	}

	records, err := wp.parser.Parse(job.FilePath, job.FileName)
	if err != nil {
		log.Printf("Error parsing file %s: %v", job.FileName, err)
		processedFile.Status = "error"
		processedFile.ErrorMsg = err.Error()

		procErr := &models.ProcessingError{
			ID:        primitive.NewObjectID(),
			FileName:  job.FileName,
			ErrorMsg:  err.Error(),
			CreatedAt: time.Now(),
		}

		if saveErr := wp.db.SaveProcessingError(ctx, procErr); saveErr != nil {
			log.Printf("Error saving processing error: %v", saveErr)
		}

		if saveErr := wp.db.SaveProcessedFile(ctx, processedFile); saveErr != nil {
			log.Printf("Error saving processed file record: %v", saveErr)
		}

		wp.moveFileToError(job.FilePath, job.FileName)
		return
	}

	if err := wp.db.SaveDeviceData(ctx, records); err != nil {
		log.Printf("Error saving data for file %s: %v", job.FileName, err)
		processedFile.Status = "error"
		processedFile.ErrorMsg = fmt.Sprintf("DB error: %v", err)

		if saveErr := wp.db.SaveProcessedFile(ctx, processedFile); saveErr != nil {
			log.Printf("Error saving processed file record: %v", saveErr)
		}
		return
	}

	unitGUIDs := wp.getUniqueUnitGUIDs(records)
	for _, unitGUID := range unitGUIDs {
		paginated, err := wp.db.GetDeviceDataByUnitGUID(ctx, unitGUID, 1, 1000)
		if err != nil {
			log.Printf("Error fetching data for unit_guid %s: %v", unitGUID, err)
			continue
		}

		reportPath, err := wp.generator.GeneratePDF(unitGUID, paginated.Data)
		if err != nil {
			log.Printf("Error generating report for unit_guid %s: %v", unitGUID, err)

			procErr := &models.ProcessingError{
				ID:        primitive.NewObjectID(),
				FileName:  job.FileName,
				UnitGUID:  unitGUID,
				ErrorMsg:  fmt.Sprintf("Report generation error: %v", err),
				CreatedAt: time.Now(),
			}

			if saveErr := wp.db.SaveProcessingError(ctx, procErr); saveErr != nil {
				log.Printf("Error saving processing error: %v", saveErr)
			}
			continue
		}

		log.Printf("Generated report for %s: %s", unitGUID, reportPath)
	}

	if err := wp.db.SaveProcessedFile(ctx, processedFile); err != nil {
		log.Printf("Error saving processed file record: %v", err)
	}

	wp.cleanupFile(job.FilePath)

	log.Printf("Successfully processed file: %s (%d records)", job.FileName, len(records))
}

func (wp *WorkerPool) getUniqueUnitGUIDs(records []*models.DeviceData) []string {
	guidMap := make(map[string]bool)
	for _, record := range records {
		guidMap[record.UnitGUID] = true
	}

	guids := make([]string, 0, len(guidMap))
	for guid := range guidMap {
		guids = append(guids, guid)
	}
	return guids
}

func (wp *WorkerPool) moveFileToError(filePath, fileName string) {
	errorDir := filepath.Join(wp.cfg.OutputDir, "errors")
	if err := os.MkdirAll(errorDir, 0755); err != nil {
		log.Printf("Error creating error directory: %v", err)
		return
	}

	destPath := filepath.Join(errorDir, fileName)
	if err := os.Rename(filePath, destPath); err != nil {
		log.Printf("Error moving file to error directory: %v", err)
	}
}

func (wp *WorkerPool) cleanupFile(filePath string) {
	archiveDir := filepath.Join(wp.cfg.OutputDir, "archive")
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		log.Printf("Error creating archive directory: %v", err)
		return
	}

	destPath := filepath.Join(archiveDir, filepath.Base(filePath))
	if err := os.Rename(filePath, destPath); err != nil {
		log.Printf("Error moving file to archive: %v", err)
	}
}
