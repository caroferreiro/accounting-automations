// WASM entry point — exposes comparePercepciones() to the browser.
// All processing happens client-side; no data leaves the browser.
//
//go:build js && wasm

package main

import (
	"bytes"
	"syscall/js"

	"accounting-automation/pkg/internal/comparador"
)

func main() {
	js.Global().Set("comparePercepciones", js.FuncOf(comparePercepciones))
	// Block forever — the WASM instance lives as long as the page is open.
	<-make(chan struct{})
}

// comparePercepciones is called from JavaScript as:
//
//	comparePercepciones(cabaUint8, bsasUint8, sistemaUint8)
//
// Returns a plain JS object: { xlsx?: Uint8Array, warnings?: string[], error?: string }
func comparePercepciones(this js.Value, args []js.Value) any {
	if len(args) != 3 {
		return errObj("se requieren 3 argumentos: caba, bsas, sistema")
	}

	cabaBytes := copyToGo(args[0])
	bsasBytes := copyToGo(args[1])
	sistemaBytes := copyToGo(args[2])

	cabaRows, cabaWarns, err := comparador.ReadFiscalRows(bytes.NewReader(cabaBytes), "901", "caba.xlsx", comparador.CABAConfig)
	if err != nil {
		return errObj(err.Error())
	}

	bsasRows, bsasWarns, err := comparador.ReadFiscalRows(bytes.NewReader(bsasBytes), "902", "bsas.xlsx", comparador.BsAsConfig)
	if err != nil {
		return errObj(err.Error())
	}

	sistemaRows, sistemaWarns, err := comparador.ReadSistemaRows(bytes.NewReader(sistemaBytes), "sistema.xlsx")
	if err != nil {
		return errObj(err.Error())
	}

	allWarnings := append(append(cabaWarns, bsasWarns...), sistemaWarns...)
	fiscalRows := append(cabaRows, bsasRows...)

	if len(fiscalRows) == 0 {
		return errObj("no se encontraron filas fiscales válidas en los archivos CABA y BsAs")
	}

	result := comparador.Compare(fiscalRows, sistemaRows)

	xlsxBytes, err := comparador.BuildXLSX(result)
	if err != nil {
		return errObj(err.Error())
	}

	obj := js.Global().Get("Object").New()

	xlsxArr := js.Global().Get("Uint8Array").New(len(xlsxBytes))
	js.CopyBytesToJS(xlsxArr, xlsxBytes)
	obj.Set("xlsx", xlsxArr)

	jsWarnings := js.Global().Get("Array").New(len(allWarnings))
	for i, w := range allWarnings {
		jsWarnings.SetIndex(i, w)
	}
	obj.Set("warnings", jsWarnings)

	return obj
}

func copyToGo(v js.Value) []byte {
	buf := make([]byte, v.Get("byteLength").Int())
	js.CopyBytesToGo(buf, v)
	return buf
}

func errObj(msg string) js.Value {
	obj := js.Global().Get("Object").New()
	obj.Set("error", msg)
	return obj
}
