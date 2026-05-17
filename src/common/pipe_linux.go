//go:build linux

// Package common proporciona utilidades compartidas por todos los servicios.
package common

import (
	"fmt"
	"os"
	"syscall"
)

// AbrirPipeLectura abre un FIFO existente en modo lectura.
// Lo crea si no existe.
func AbrirPipeLectura(ruta string) (*os.File, error) {
	if err := crearFIFO(ruta); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(ruta, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("AbrirPipeLectura: %w", err)
	}
	return f, nil
}

// AbrirPipeEscritura abre un FIFO existente en modo escritura.
// Lo crea si no existe.
func AbrirPipeEscritura(ruta string) (*os.File, error) {
	if err := crearFIFO(ruta); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(ruta, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("AbrirPipeEscritura: %w", err)
	}
	return f, nil
}

// crearFIFO crea un FIFO en la ruta dada si no existe.
func crearFIFO(ruta string) error {
	if _, err := os.Stat(ruta); os.IsNotExist(err) {
		if err := syscall.Mkfifo(ruta, 0666); err != nil {
			return fmt.Errorf("crearFIFO: %w", err)
		}
	}
	return nil
}
