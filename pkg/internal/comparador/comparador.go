package comparador

import (
	"bytes"
	"fmt"

	"github.com/shopspring/decimal"
	"github.com/xuri/excelize/v2"
)

func makeKey(jur, cuit, fecha string, importe decimal.Decimal) string {
	return fmt.Sprintf("%s|%s|%s|%s", jur, cuit, fecha, importe.StringFixed(2))
}

// Compare compares fiscal rows against sistema rows.
// Handles duplicates by frequency: if fiscal has N identical rows and
// sistema has M, min(N,M) rows get "SI" and the rest get "NO".
func Compare(fiscal []FiscalRow, sistema []SistemaRow) CompareResult {
	// Build frequency map for sistema
	freq := make(map[string]int, len(sistema))
	for _, r := range sistema {
		freq[makeKey(r.Jurisdiccion, r.CUIT, r.Fecha, r.Importe)]++
	}

	remaining := make(map[string]int, len(freq))
	for k, v := range freq {
		remaining[k] = v
	}

	rows := make([]ResultRow, 0, len(fiscal))
	for _, r := range fiscal {
		k := makeKey(r.Jurisdiccion, r.CUIT, r.Fecha, r.Importe)
		coincide := "NO"
		if remaining[k] > 0 {
			remaining[k]--
			coincide = "SI"
		}
		rows = append(rows, ResultRow{
			Jurisdiccion: r.Jurisdiccion,
			CUIT:         r.CUIT,
			Fecha:        r.Fecha,
			Importe:      r.Importe,
			Coincide:     coincide,
		})
	}

	// Collect sistema rows that were not matched by any fiscal row
	processed := make(map[string]bool)
	var sinMatch []SistemaRow
	for _, r := range sistema {
		k := makeKey(r.Jurisdiccion, r.CUIT, r.Fecha, r.Importe)
		if !processed[k] {
			processed[k] = true
			for i := 0; i < remaining[k]; i++ {
				sinMatch = append(sinMatch, r)
			}
		}
	}

	return CompareResult{Rows: rows, SinMatch: sinMatch}
}

// BuildXLSX generates the result Excel workbook as a byte slice.
// Sheet "Comparacion": all fiscal rows with Coincide SI/NO.
// Sheet "Sistema sin match" (optional): sistema rows without a matching fiscal row.
func BuildXLSX(result CompareResult) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	const sheet1 = "Comparacion"
	f.SetSheetName("Sheet1", sheet1)

	headers1 := []string{"Jurisdicción", "CUIT", "Fecha", "Importe", "Coincide"}
	for col, h := range headers1 {
		f.SetCellValue(sheet1, coord(col+1, 1), h)
	}
	for i, r := range result.Rows {
		row := i + 2
		f.SetCellValue(sheet1, coord(1, row), r.Jurisdiccion)
		f.SetCellValue(sheet1, coord(2, row), r.CUIT)
		f.SetCellValue(sheet1, coord(3, row), r.Fecha)
		f.SetCellValue(sheet1, coord(4, row), r.Importe.InexactFloat64())
		f.SetCellValue(sheet1, coord(5, row), r.Coincide)
	}

	if len(result.SinMatch) > 0 {
		const sheet2 = "Sistema sin match"
		f.NewSheet(sheet2)
		headers2 := []string{"Jurisdicción", "CUIT", "Fecha", "Importe"}
		for col, h := range headers2 {
			f.SetCellValue(sheet2, coord(col+1, 1), h)
		}
		for i, r := range result.SinMatch {
			row := i + 2
			f.SetCellValue(sheet2, coord(1, row), r.Jurisdiccion)
			f.SetCellValue(sheet2, coord(2, row), r.CUIT)
			f.SetCellValue(sheet2, coord(3, row), r.Fecha)
			f.SetCellValue(sheet2, coord(4, row), r.Importe.InexactFloat64())
		}
	}

	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		return nil, fmt.Errorf("error generando xlsx: %w", err)
	}
	return buf.Bytes(), nil
}

func coord(col, row int) string {
	name, _ := excelize.CoordinatesToCellName(col, row)
	return name
}
