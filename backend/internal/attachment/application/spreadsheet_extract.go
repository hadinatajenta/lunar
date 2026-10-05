package application

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/extrame/xls"
	"github.com/xuri/excelize/v2"
)

const (
	maxLegacySpreadsheetRows = 2000
	legacySpreadsheetName    = "legacy spreadsheet"
	cellSeparator            = "\t"
	rowSeparator             = "\n"
)

func extractXLSXText(content []byte) (text string, err error) {
	workbook, err := excelize.OpenReader(bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}
	defer func() {
		if closeErr := workbook.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("%w: %v", ErrUnreadableDocument, closeErr)
		}
	}()

	var builder strings.Builder
	for _, sheetName := range workbook.GetSheetList() {
		if err := appendXLSXSheet(&builder, workbook, sheetName); err != nil {
			return "", err
		}
		if builder.Len() > MaxExtractedCharacters {
			break
		}
	}

	return capExtractedText(builder.String()), nil
}

func appendXLSXSheet(builder *strings.Builder, workbook *excelize.File, sheetName string) (err error) {
	rows, err := workbook.Rows(sheetName)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil && err == nil {
			err = fmt.Errorf("%w: %v", ErrUnreadableDocument, closeErr)
		}
	}()

	appendSpreadsheetLabel(builder, sheetName)
	for rows.Next() {
		columns, err := rows.Columns()
		if err != nil {
			return fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
		}
		appendSpreadsheetRow(builder, columns)
		if builder.Len() > MaxExtractedCharacters {
			break
		}
	}
	if err := rows.Error(); err != nil {
		return fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}
	return nil
}

func extractXLSText(content []byte) (string, error) {
	workbook, err := xls.OpenReader(bytes.NewReader(content), "utf-8")
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrUnreadableDocument, err)
	}

	var builder strings.Builder
	appendSpreadsheetLabel(&builder, legacySpreadsheetName)
	for _, row := range workbook.ReadAllCells(maxLegacySpreadsheetRows) {
		appendSpreadsheetRow(&builder, row)
		if builder.Len() > MaxExtractedCharacters {
			break
		}
	}

	return capExtractedText(builder.String()), nil
}

func appendSpreadsheetLabel(builder *strings.Builder, sheetName string) {
	builder.WriteString("[sheet] ")
	builder.WriteString(sheetName)
	builder.WriteString(rowSeparator)
}

func appendSpreadsheetRow(builder *strings.Builder, columns []string) {
	builder.WriteString(strings.Join(columns, cellSeparator))
	builder.WriteString(rowSeparator)
}
