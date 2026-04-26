# Comparador de Percepciones

Herramienta local para comparar percepciones fiscales (CABA y Provincia de Buenos Aires) contra el sistema contable.

## Requisitos

- Go 1.24+

## Cómo correr

Desde la raíz del repositorio:

```bash
go run ./pkg/cmd/server/
```

Luego abrir http://localhost:8080 en el navegador.

## Cómo correr los tests

```bash
go test ./...
```

## Archivos de entrada

| Archivo | Formato | Columnas relevantes |
|---------|---------|-------------------|
| CABA | `.xlsx`, sin cabecera | A=CUIT, F=Fecha, J=Importe |
| Prov. Buenos Aires | `.xlsx`, sin cabecera | A=CUIT, F=Fecha, J=Importe |
| Sistema contable | `.xlsx`, sin cabecera | 0=Jurisdicción, 1=CUIT, 2=Fecha, 7=Importe percibido |

Solo se procesan filas del sistema con jurisdicción `901` (CABA) o `902` (BsAs).

## Archivo de salida

El resultado es un `.xlsx` con:

- **Hoja `Comparacion`**: todas las filas fiscales con columna `Coincide` (`SI`/`NO`).
- **Hoja `Sistema sin match`** *(si aplica)*: filas del sistema sin fila fiscal equivalente.

## Estructura del proyecto

```
pkg/
  cmd/server/         servidor HTTP (entry point)
  internal/comparador/
    model.go          tipos de datos
    normalizer.go     normalización de CUIT, fecha e importe
    reader.go         lectura de archivos Excel
    comparador.go     lógica de comparación y generación de xlsx
    comparador_test.go
web/
  index.html          interfaz web
```
