package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"accounting-automation/pkg/internal/comparador"
)

func main() {
	mux := http.NewServeMux()
	mux.Handle("GET /", http.FileServer(http.Dir("web")))
	mux.HandleFunc("POST /compare", handleCompare)

	addr := ":8080"
	fmt.Printf("Servidor iniciado en http://localhost%s\n", addr)
	log.Fatal(http.ListenAndServe(addr, mux))
}


func handleCompare(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(64 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "error leyendo form: "+err.Error())
		return
	}

	type fiscalSpec struct {
		field        string
		label        string
		jurisdiccion string
		cfg          comparador.FiscalFileConfig
	}

	var allFiscal []comparador.FiscalRow
	var allWarnings []string

	for _, spec := range []fiscalSpec{
		{"caba", "CABA", "901", comparador.CABAConfig},
		{"bsas", "Prov. Buenos Aires", "902", comparador.BsAsConfig},
	} {
		f, fh, err := r.FormFile(spec.field)
		if err != nil {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("archivo %s faltante o inválido: %v", spec.label, err))
			return
		}
		defer f.Close()

		rows, warns, err := comparador.ReadFiscalRows(f, spec.jurisdiccion, fh.Filename, spec.cfg)
		if err != nil {
			writeError(w, http.StatusUnprocessableEntity, err.Error())
			return
		}
		allFiscal = append(allFiscal, rows...)
		allWarnings = append(allWarnings, warns...)
	}

	sistemaFile, sistemaFH, err := r.FormFile("sistema")
	if err != nil {
		writeError(w, http.StatusBadRequest, "archivo sistema faltante o inválido: "+err.Error())
		return
	}
	defer sistemaFile.Close()

	sistemaRows, warns, err := comparador.ReadSistemaRows(sistemaFile, sistemaFH.Filename)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	allWarnings = append(allWarnings, warns...)

	if len(allFiscal) == 0 {
		writeError(w, http.StatusUnprocessableEntity, "no se encontraron filas fiscales válidas en los archivos CABA y BsAs")
		return
	}

	result := comparador.Compare(allFiscal, sistemaRows)

	xlsxBytes, err := comparador.BuildXLSX(result)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	filename := fmt.Sprintf("comparacion_percepciones_%s.xlsx", time.Now().Format("2006-01-02"))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, filename))
	if len(allWarnings) > 0 {
		warningsJSON, _ := json.Marshal(allWarnings)
		w.Header().Set("X-Warnings", url.QueryEscape(string(warningsJSON)))
	}
	w.Write(xlsxBytes)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}
