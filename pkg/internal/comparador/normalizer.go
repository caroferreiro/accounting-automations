package comparador

import (
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/shopspring/decimal"
)

// NormalizeCUIT removes spaces, dashes, and dots.
// Handles scientific notation that Excel may produce for large numbers.
func NormalizeCUIT(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	// Handle Excel scientific notation (e.g. "3.06772E+10")
	if strings.ContainsAny(s, "eE") {
		if f, err := strconv.ParseFloat(s, 64); err == nil {
			s = fmt.Sprintf("%.0f", f)
		}
	}
	var b strings.Builder
	for _, r := range s {
		if r != ' ' && r != '-' && r != '.' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// NormalizeDate parses a date value to YYYY-MM-DD.
// Supports:
//   - YYYY-MM-DD strings
//   - M/D/YY and M/D/YYYY (Excel US display format observed in fiscal files)
//   - DD/MM/YYYY
//   - Excel serial numbers (stored as float strings)
func NormalizeDate(s string) (string, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", fmt.Errorf("fecha vacía")
	}

	// Already YYYY-MM-DD
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return t.Format("2006-01-02"), nil
	}

	// DD/MM/YYYY — must be tried before M/D/YY to avoid ambiguity with zero-padded dates
	if t, err := time.Parse("02/01/2006", s); err == nil {
		return t.Format("2006-01-02"), nil
	}

	// MM-DD-YY / MM-DD-YYYY — excelize internal date formatting (dashes, zero-padded)
	for _, layout := range []string{"01-02-06", "01-02-2006"} {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("2006-01-02"), nil
		}
	}

	// M/D/YY — Excel US display format observed in fiscal files ("3/2/26" = March 2, 2026)
	if t, err := time.Parse("1/2/06", s); err == nil {
		return t.Format("2006-01-02"), nil
	}

	// Excel serial number stored as float string ("45700", "45700.5", etc.)
	if isNumericString(s) {
		if serial, err := strconv.ParseFloat(s, 64); err == nil {
			return excelSerialToDate(serial), nil
		}
	}

	return "", fmt.Errorf("no se pudo parsear la fecha: %q", s)
}

// NormalizeImporte parses an amount to a Decimal rounded to 2 decimal places.
// Supports:
//   - Plain numbers: "100.50"
//   - Comma decimal: "100,50"
//   - Argentine thousands format: "1.266,50"
func NormalizeImporte(s string) (decimal.Decimal, error) {
	s = strings.TrimSpace(s)
	// Strip currency symbols and spaces
	s = strings.Map(func(r rune) rune {
		if r == '$' || unicode.IsSpace(r) {
			return -1
		}
		return r
	}, s)
	if s == "" {
		return decimal.Zero, fmt.Errorf("importe vacío")
	}

	lastDot := strings.LastIndex(s, ".")
	lastComma := strings.LastIndex(s, ",")

	switch {
	case lastDot >= 0 && lastComma >= 0:
		if lastComma > lastDot {
			// Argentine format: dot=thousands, comma=decimal ("1.266,50" → 1266.50)
			s = strings.ReplaceAll(s, ".", "")
			s = strings.ReplaceAll(s, ",", ".")
		} else {
			// US format: comma=thousands, dot=decimal ("1,266.50" → 1266.50)
			s = strings.ReplaceAll(s, ",", "")
		}
	case lastComma >= 0:
		// Only comma: treat as decimal separator ("100,50")
		s = strings.ReplaceAll(s, ",", ".")
	}

	d, err := decimal.NewFromString(s)
	if err != nil {
		return decimal.Zero, fmt.Errorf("no se pudo parsear el importe: %q", s)
	}
	return d.Round(2), nil
}

// excelSerialToDate converts an Excel serial date number to YYYY-MM-DD.
// Excel serial 1 = 1900-01-01. The value 25569 = 1970-01-01 (Unix epoch).
// Uses the standard correction for Excel's spurious leap year in 1900.
func excelSerialToDate(serial float64) string {
	epoch := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	d := epoch.AddDate(0, 0, int(serial))
	return d.Format("2006-01-02")
}

func isNumericString(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) && r != '.' && r != '-' {
			return false
		}
	}
	return true
}
