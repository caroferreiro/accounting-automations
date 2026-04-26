package comparador

import "github.com/shopspring/decimal"

// FiscalFileConfig defines which column indices hold CUIT, Fecha, and Importe
// in a fiscal file. Different jurisdictions may use different layouts.
type FiscalFileConfig struct {
	ColCUIT    int
	ColFecha   int
	ColImporte int
}

// CABAConfig: A=CUIT(0), F=Fecha(5), J=Importe(9)
var CABAConfig = FiscalFileConfig{ColCUIT: 0, ColFecha: 5, ColImporte: 9}

// BsAsConfig: A=CUIT(0), B=Fecha(1), K=Importe(10)
var BsAsConfig = FiscalFileConfig{ColCUIT: 0, ColFecha: 1, ColImporte: 10}

type FiscalRow struct {
	Jurisdiccion string
	CUIT         string
	Fecha        string // YYYY-MM-DD
	Importe      decimal.Decimal
}

type SistemaRow struct {
	Jurisdiccion string
	CUIT         string
	Fecha        string // YYYY-MM-DD
	Importe      decimal.Decimal
}

type ResultRow struct {
	Jurisdiccion string
	CUIT         string
	Fecha        string
	Importe      decimal.Decimal
	Coincide     string // "SI" or "NO"
}

type CompareResult struct {
	Rows     []ResultRow
	SinMatch []SistemaRow
}
