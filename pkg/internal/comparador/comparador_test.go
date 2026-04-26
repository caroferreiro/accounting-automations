package comparador

import (
	"strings"
	"testing"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

// ---- Normalization tests ----

func TestNormalizeCUIT(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"30-12345678-9", "30123456789"},
		{"30.123.456.789", "30123456789"},
		{"30 123 456 789", "30123456789"},
		{"30123456789", "30123456789"},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, NormalizeCUIT(c.input), "input: %q", c.input)
	}
}

func TestNormalizeDate(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"2026-03-01", "2026-03-01"},
		{"3/1/26", "2026-03-01"},     // M/D/YY (Excel US format)
		{"3/16/26", "2026-03-16"},    // M/D/YY
		{"01/03/2026", "2026-03-01"}, // DD/MM/YYYY
		{"46082", "2026-03-01"},      // Excel serial for 2026-03-01
	}
	for _, c := range cases {
		got, err := NormalizeDate(c.input)
		require.NoError(t, err, "input: %q", c.input)
		assert.Equal(t, c.want, got, "input: %q", c.input)
	}
}

func TestNormalizeDateError(t *testing.T) {
	_, err := NormalizeDate("")
	assert.Error(t, err)
	_, err = NormalizeDate("no-es-una-fecha")
	assert.Error(t, err)
}

func TestNormalizeImporte(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"100.50", "100.50"},
		{"100,50", "100.50"},
		{"1.266,50", "1266.50"},  // Argentine thousands format
		{"1,266.50", "1266.50"},  // US thousands format (dot=decimal)
		{"23,11", "23.11"},
		{"0", "0.00"},
	}
	for _, c := range cases {
		got, err := NormalizeImporte(c.input)
		require.NoError(t, err, "input: %q", c.input)
		assert.Equal(t, c.want, got.StringFixed(2), "input: %q", c.input)
	}
}

// ---- Compare tests ----

func d(s string) decimal.Decimal { return decimal.RequireFromString(s) }

func row901(cuit, fecha, importe string) FiscalRow {
	return FiscalRow{Jurisdiccion: "901", CUIT: cuit, Fecha: fecha, Importe: d(importe)}
}
func row902(cuit, fecha, importe string) FiscalRow {
	return FiscalRow{Jurisdiccion: "902", CUIT: cuit, Fecha: fecha, Importe: d(importe)}
}
func srow(jur, cuit, fecha, importe string) SistemaRow {
	return SistemaRow{Jurisdiccion: jur, CUIT: cuit, Fecha: fecha, Importe: d(importe)}
}

// Test 1: coincidencia exacta CABA (901)
func TestCompare_ExactMatchCABA(t *testing.T) {
	fiscal := []FiscalRow{row901("30123456789", "2026-03-01", "100.50")}
	sistema := []SistemaRow{srow("901", "30123456789", "2026-03-01", "100.50")}

	result := Compare(fiscal, sistema)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, "SI", result.Rows[0].Coincide)
	assert.Empty(t, result.SinMatch)
}

// Test 2: coincidencia exacta BsAs (902)
func TestCompare_ExactMatchBsAs(t *testing.T) {
	fiscal := []FiscalRow{row902("20987654321", "2026-03-15", "250.00")}
	sistema := []SistemaRow{srow("902", "20987654321", "2026-03-15", "250.00")}

	result := Compare(fiscal, sistema)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, "SI", result.Rows[0].Coincide)
}

// Test 3: diferencia por importe → NO
func TestCompare_MismatchImporte(t *testing.T) {
	fiscal := []FiscalRow{row901("30123456789", "2026-03-01", "100.50")}
	sistema := []SistemaRow{srow("901", "30123456789", "2026-03-01", "200.00")}

	result := Compare(fiscal, sistema)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, "NO", result.Rows[0].Coincide)
}

// Test 4: diferencia por fecha → NO
func TestCompare_MismatchFecha(t *testing.T) {
	fiscal := []FiscalRow{row901("30123456789", "2026-03-01", "100.50")}
	sistema := []SistemaRow{srow("901", "30123456789", "2026-03-02", "100.50")}

	result := Compare(fiscal, sistema)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, "NO", result.Rows[0].Coincide)
}

// Test 5: diferencia por CUIT → NO
func TestCompare_MismatchCUIT(t *testing.T) {
	fiscal := []FiscalRow{row901("30123456789", "2026-03-01", "100.50")}
	sistema := []SistemaRow{srow("901", "30999999999", "2026-03-01", "100.50")}

	result := Compare(fiscal, sistema)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, "NO", result.Rows[0].Coincide)
}

// Test 6: duplicados — fiscal tiene 2 iguales, sistema tiene 1 → 1 SI + 1 NO
func TestCompare_Duplicates(t *testing.T) {
	fiscal := []FiscalRow{
		row901("30123456789", "2026-03-01", "100.50"),
		row901("30123456789", "2026-03-01", "100.50"),
	}
	sistema := []SistemaRow{
		srow("901", "30123456789", "2026-03-01", "100.50"),
	}

	result := Compare(fiscal, sistema)

	require.Len(t, result.Rows, 2)
	coincides := []string{result.Rows[0].Coincide, result.Rows[1].Coincide}
	assert.Contains(t, coincides, "SI")
	assert.Contains(t, coincides, "NO")
}

// Test 7: normalización — CUIT con guiones vs sin guiones, importe con coma vs número
func TestCompare_Normalization(t *testing.T) {
	// Simulates what ReadFiscalRows would return after normalization
	fiscalCUIT := NormalizeCUIT("30-12345678-9")
	importeFiscal, err := NormalizeImporte("100,50")
	require.NoError(t, err)
	fechaFiscal, err := NormalizeDate("3/1/26")
	require.NoError(t, err)

	sistemaCUIT := NormalizeCUIT("30123456789")
	importeSistema, err := NormalizeImporte("100.50")
	require.NoError(t, err)
	fechaSistema, err := NormalizeDate("2026-03-01")
	require.NoError(t, err)

	fiscal := []FiscalRow{{Jurisdiccion: "901", CUIT: fiscalCUIT, Fecha: fechaFiscal, Importe: importeFiscal}}
	sistema := []SistemaRow{{Jurisdiccion: "901", CUIT: sistemaCUIT, Fecha: fechaSistema, Importe: importeSistema}}

	result := Compare(fiscal, sistema)

	require.Len(t, result.Rows, 1)
	assert.Equal(t, "SI", result.Rows[0].Coincide, "CUIT/importe/fecha normalizados deben coincidir")
}

// Test 8: sin match en sistema aparece en SinMatch
func TestCompare_SistemaWithoutFiscalMatch(t *testing.T) {
	fiscal := []FiscalRow{row901("30123456789", "2026-03-01", "100.50")}
	sistema := []SistemaRow{
		srow("901", "30123456789", "2026-03-01", "100.50"),
		srow("901", "30999999999", "2026-03-02", "200.00"), // no fiscal equivalent
	}

	result := Compare(fiscal, sistema)

	require.Len(t, result.SinMatch, 1)
	assert.Equal(t, "30999999999", result.SinMatch[0].CUIT)
}

// Test 9: filas de ambas jurisdicciones en la misma comparación
func TestCompare_BothJurisdictions(t *testing.T) {
	fiscal := []FiscalRow{
		row901("30111111111", "2026-03-01", "50.00"),
		row902("30222222222", "2026-03-05", "75.00"),
	}
	sistema := []SistemaRow{
		srow("901", "30111111111", "2026-03-01", "50.00"),
		srow("902", "30222222222", "2026-03-05", "75.00"),
	}

	result := Compare(fiscal, sistema)

	require.Len(t, result.Rows, 2)
	assert.Equal(t, "SI", result.Rows[0].Coincide)
	assert.Equal(t, "SI", result.Rows[1].Coincide)
}

// ---- ReadFiscalRows tests ----

func TestReadFiscalRows(t *testing.T) {
	// Synthetic xlsx: CUIT in col A, Fecha in col F, Importe in col J
	xlsxData := buildSyntheticFiscalXLSX(t, [][]string{
		{"30123456789", "", "", "", "", "3/1/26", "", "", "", "100.50"},
		{"30-987654321", "", "", "", "", "3/15/26", "", "", "", "1.266,50"},
		{"", "", "", "", "", "", "", "", "", ""},
	})

	rows, warnings, err := ReadFiscalRows(strings.NewReader(xlsxData), "901", "test.xlsx", CABAConfig)
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, rows, 2)

	assert.Equal(t, "30123456789", rows[0].CUIT)
	assert.Equal(t, "2026-03-01", rows[0].Fecha)
	assert.Equal(t, "100.50", rows[0].Importe.StringFixed(2))

	assert.Equal(t, "30987654321", rows[1].CUIT)
	assert.Equal(t, "2026-03-15", rows[1].Fecha)
	assert.Equal(t, "1266.50", rows[1].Importe.StringFixed(2))
}

func TestReadSistemaRows(t *testing.T) {
	// Synthetic xlsx: 8 columns, col 0=Jur, 1=CUIT, 2=Fecha, 7=Importe
	xlsxData := buildSyntheticSistemaXLSX(t, [][]string{
		{"901", "30123456789", "2026-03-01", "", "", "", "", "100.50"},
		{"902", "30987654321", "2026-03-05", "", "", "", "", "75.00"},
		{"999", "30000000000", "2026-03-01", "", "", "", "", "50.00"}, // ignored: unknown jur
		{"", "", "", "", "", "", "", ""},                               // ignored: empty
	})

	rows, warnings, err := ReadSistemaRows(strings.NewReader(xlsxData), "test.xlsx")
	require.NoError(t, err)
	assert.Empty(t, warnings)
	require.Len(t, rows, 2)
	assert.Equal(t, "901", rows[0].Jurisdiccion)
	assert.Equal(t, "902", rows[1].Jurisdiccion)
}

// ---- helpers for building synthetic xlsx in tests ----

func buildSyntheticFiscalXLSX(t *testing.T, data [][]string) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Sheet1"
	for rowIdx, row := range data {
		for colIdx, val := range row {
			c, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			f.SetCellValue(sheet, c, val)
		}
	}
	var buf strings.Builder
	_ = f.Write(writerFunc(func(b []byte) (int, error) { buf.Write(b); return len(b), nil }))
	return buf.String()
}

func buildSyntheticSistemaXLSX(t *testing.T, data [][]string) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	sheet := "Sheet1"
	for rowIdx, row := range data {
		for colIdx, val := range row {
			c, _ := excelize.CoordinatesToCellName(colIdx+1, rowIdx+1)
			f.SetCellValue(sheet, c, val)
		}
	}
	var buf strings.Builder
	_ = f.Write(writerFunc(func(b []byte) (int, error) { buf.Write(b); return len(b), nil }))
	return buf.String()
}

type writerFunc func([]byte) (int, error)

func (wf writerFunc) Write(b []byte) (int, error) { return wf(b) }
