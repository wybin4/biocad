package generator

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/jung-kurt/gofpdf/v2"
	"github.com/tsv-processor/internal/models"
)

type ReportGenerator struct {
	outputDir string
}

func NewReportGenerator(outputDir string) *ReportGenerator {
	return &ReportGenerator{
		outputDir: outputDir,
	}
}

func (g *ReportGenerator) GeneratePDF(unitGUID string, data []models.DeviceData) (string, error) {
	pdf := gofpdf.New("L", "mm", "A4", "")
	pdf.AddPage()

	pdf.AddUTF8Font("DejaVu", "", "./fonts/DejaVuSans.ttf")
	pdf.AddUTF8Font("DejaVu", "B", "./fonts/DejaVuSans-Bold.ttf")
	pdf.SetFont("DejaVu", "", 12)

	pdf.SetFont("DejaVu", "B", 16)
	pdf.CellFormat(277, 10, fmt.Sprintf("GUID устройства: %s", unitGUID), "", 0, "C", false, 0, "")
	pdf.Ln(12)

	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(277, 6, fmt.Sprintf("Дата отчета: %s", time.Now().Format("2006-01-02 15:04:05")), "", 0, "C", false, 0, "")
	pdf.Ln(15)

	if len(data) > 0 {
		pdf.SetFont("DejaVu", "B", 14)
		pdf.Cell(277, 8, "Информация об устройстве")
		pdf.Ln(10)

		pdf.SetFont("DejaVu", "", 11)
		inventory := data[0].Inventory
		pdf.Cell(277, 7, fmt.Sprintf("Инвентарный номер: %s", inventory))
		pdf.Ln(7)

		pdf.Cell(277, 7, fmt.Sprintf("Всего записей: %d", len(data)))
		pdf.Ln(12)
	}

	pdf.SetFont("DejaVu", "B", 14)
	pdf.Cell(277, 8, "Статистика сообщений")
	pdf.Ln(10)

	pdf.SetFont("DejaVu", "", 11)

	classCount := make(map[string]int)
	uniqueMsgIDs := make(map[string]bool)
	for _, d := range data {
		classCount[d.Class]++
		uniqueMsgIDs[d.MsgID] = true
	}

	pdf.Cell(277, 7, fmt.Sprintf("Уникальных ID сообщений: %d", len(uniqueMsgIDs)))
	pdf.Ln(7)

	for class, count := range classCount {
		className := class
		switch class {
		case "alarm":
			className = "Авария"
			pdf.SetTextColor(255, 0, 0)
		case "warning":
			className = "Предупреждение"
			pdf.SetTextColor(255, 165, 0)
		case "working":
			className = "Работа"
			pdf.SetTextColor(0, 128, 0)
		case "waiting":
			className = "Ожидание"
			pdf.SetTextColor(128, 128, 128)
		}

		pdf.Cell(277, 7, fmt.Sprintf("• %s: %d", className, count))
		pdf.Ln(7)
		pdf.SetTextColor(0, 0, 0)
	}
	pdf.Ln(8)

	pdf.SetFont("DejaVu", "B", 14)
	pdf.Cell(277, 8, "Детальная информация")
	pdf.Ln(12)

	pdf.SetFont("DejaVu", "B", 8)
	pdf.SetFillColor(240, 240, 240)

	colWidths := []float64{8, 60, 60, 20, 15, 15, 99}
	headers := []string{"№", "ID сообщения", "Текст", "Класс", "Уровень", "Зона", "Адрес"}

	for i, header := range headers {
		pdf.CellFormat(colWidths[i], 8, header, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("DejaVu", "", 7)

	seen := make(map[string]bool)
	uniqueData := []models.DeviceData{}

	for _, d := range data {
		key := fmt.Sprintf("%d-%s-%s", d.RowNum, d.MsgID, d.Addr)
		if !seen[key] {
			seen[key] = true
			uniqueData = append(uniqueData, d)
		}
	}

	for i, d := range uniqueData {
		if i >= 30 {
			pdf.SetFont("DejaVu", "I", 7)
			pdf.CellFormat(277, 5, "... и еще записи", "", 0, "C", false, 0, "")
			pdf.Ln(5)
			break
		}

		msgID := d.MsgID
		textMsg := d.Text
		addr := d.Addr

		className := d.Class
		switch d.Class {
		case "alarm":
			className = "Авария"
			pdf.SetTextColor(255, 0, 0)
		case "warning":
			className = "Предупр."
			pdf.SetTextColor(255, 165, 0)
		case "working":
			className = "Работа"
			pdf.SetTextColor(0, 128, 0)
		case "waiting":
			className = "Ожидание"
			pdf.SetTextColor(128, 128, 128)
		}

		pdf.CellFormat(colWidths[0], 6, fmt.Sprintf("%d", d.RowNum), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[1], 6, msgID, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[2], 6, textMsg, "1", 0, "L", false, 0, "")
		pdf.CellFormat(colWidths[3], 6, className, "1", 0, "C", false, 0, "")
		pdf.SetTextColor(0, 0, 0)
		pdf.CellFormat(colWidths[4], 6, fmt.Sprintf("%d", d.Level), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[5], 6, d.Area, "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[6], 6, addr, "1", 0, "L", false, 0, "")
		pdf.Ln(-1)
	}

	pdf.Ln(5)
	pdf.SetFont("DejaVu", "B", 10)
	pdf.CellFormat(277, 7, fmt.Sprintf("Всего записей: %d", len(uniqueData)), "", 0, "R", false, 0, "")

	fileName := fmt.Sprintf("device_%s_%s.pdf", unitGUID, time.Now().Format("20060102_150405"))
	filePath := filepath.Join(g.outputDir, fileName)

	err := pdf.OutputFileAndClose(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to save PDF: %w", err)
	}

	return filePath, nil
}
