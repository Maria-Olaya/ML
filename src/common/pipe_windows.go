//go:build windows

// Package common proporciona utilidades compartidas.
// Implementación temporal de pipes para Windows usando named pipes full-duplex.
package common

import (
	"fmt"
	"os"
)

// AbrirPipes abre un único named pipe full-duplex para comunicación.
// En Windows, se usa un solo pipe que sirve para leer y escribir.
// Los dos parámetros se ignoran (se usa pipePeticiones como nombre único).
func AbrirPipes(pipePeticiones, pipeRespuestas string) (*os.File, *os.File, error) {
	// En full-duplex, solo necesitamos un pipe.
	// Se abre en modo lectura/escritura.
	// Nota: El pipe debe haber sido creado por el servidor (ctrllt).
	// Para pruebas manuales, puedes crearlo con: $pipe = \\.\pipe\gesprog_in
	// y luego conectarte como cliente.

	// Abrir el pipe en modo lectura/escritura.
	// En Windows, para conectarse a un named pipe existente se usa CreateFile,
	// que se puede hacer con os.OpenFile con los modos adecuados.
	pipe, err := os.OpenFile(pipePeticiones, os.O_RDWR, os.ModeNamedPipe)
	if err != nil {
		return nil, nil, fmt.Errorf("AbrirPipes: no se pudo abrir el pipe: %w", err)
	}
	// En full-duplex, el mismo descriptor sirve para leer y escribir.
	// Devolvemos el mismo *File para ambos.
	return pipe, pipe, nil
}
