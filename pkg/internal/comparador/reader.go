package comparador

import (
	"fmt"
	"io"
	"strings"

	"github.com/xuri/excelize/v2"
)

// Column indices in sistema file — no header row.
const (
	sistemaColJurisdiccion = 0
	sistemaColCUIT         = 1
	sistemaColFecha        = 2
	sistemaColImporte      = 7
	sistemaMinCols         = 8
)

// ReadFiscalRows reads a fiscal file (CABA or BsAs).
// No header row. Column positions are defined by cfg (use CABAConfig or BsAsConfig).
// Returns parsed rows, non-fatal row-level warnings, and a fatal error if the file can't be opened.
func ReadFiscalRows(r io.Reader, jurisdiccion, filename string, cfg FiscalFileConfig) ([]FiscalRow, []string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudo leer %q: %w", filename, err)
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		return nil, nil, fmt.Errorf("error leyendo hoja de %q: %w", filename, err)
	}

	var result []FiscalRow
	var warnings []string

	for i, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		rawCUIT := cell(row, cfg.ColCUIT)
		if rawCUIT == "" {
			continue
		}
		cuit := NormalizeCUIT(rawCUIT)
		if cuit == "" {
			continue
		}

		fecha, err := NormalizeDate(cell(row, cfg.ColFecha))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s fila %d: %v", filename, i+1, err))
			continue
		}
		importe, err := NormalizeImporte(cell(row, cfg.ColImporte))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s fila %d: %v", filename, i+1, err))
			continue
		}

		result = append(result, FiscalRow{
			Jurisdiccion: jurisdiccion,
			CUIT:         cuit,
			Fecha:        fecha,
			Importe:      importe,
		})
	}

	return result, warnings, nil
}

// ReadSistemaRows reads a sistema contable file.
// No header row. Column order: 0=Jurisdiccion, 1=CUIT, 2=Fecha, 7=Importe percibido.
// Only includes rows with jurisdiccion 901 or 902.
func ReadSistemaRows(r io.Reader, filename string) ([]SistemaRow, []string, error) {
	f, err := excelize.OpenReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("no se pudo leer %q: %w", filename, err)
	}
	defer f.Close()

	rows, err := f.GetRows(f.GetSheetName(0))
	if err != nil {
		return nil, nil, fmt.Errorf("error leyendo hoja de %q: %w", filename, err)
	}

	var result []SistemaRow
	var warnings []string

	for i, row := range rows {
		if isEmptyRow(row) {
			continue
		}
		if len(row) < sistemaMinCols {
			warnings = append(warnings, fmt.Sprintf("%s fila %d: tiene %d columnas (se requieren %d), ignorada", filename, i+1, len(row), sistemaMinCols))
			continue
		}

		jur := strings.TrimSpace(row[sistemaColJurisdiccion])
		if jur != "901" && jur != "902" {
			continue
		}

		cuit := NormalizeCUIT(cell(row, sistemaColCUIT))
		if cuit == "" {
			continue
		}

		fecha, err := NormalizeDate(cell(row, sistemaColFecha))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s fila %d: %v", filename, i+1, err))
			continue
		}
		importe, err := NormalizeImporte(cell(row, sistemaColImporte))
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("%s fila %d: %v", filename, i+1, err))
			continue
		}

		result = append(result, SistemaRow{
			Jurisdiccion: jur,
			CUIT:         cuit,
			Fecha:        fecha,
			Importe:      importe,
		})
	}

	return result, warnings, nil
}

func cell(row []string, idx int) string {
	if idx >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[idx])
}

func isEmptyRow(row []string) bool {
	for _, c := range row {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}
