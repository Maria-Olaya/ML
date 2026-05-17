//go:build windows

// Package common proporciona utilidades compartidas por todos los servicios.
package common

import (
	"fmt"
	"os"
)

// AbrirPipeLectura abre un named pipe en modo lectura.
func AbrirPipeLectura(ruta string) (*os.File, error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("AbrirPipeLectura: %w", err)
	}
	return f, nil
}

// AbrirPipeEscritura abre un named pipe en modo escritura.
func AbrirPipeEscritura(ruta string) (*os.File, error) {
	f, err := os.OpenFile(ruta, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("AbrirPipeEscritura: %w", err)
	}
	return f, nil
}
