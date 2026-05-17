//go:build linux

// Package common proporciona utilidades compartidas.
// Implementación temporal de pipes para Linux usando FIFOs.
package common

import (
	"fmt"
	"os"
	"syscall"
)

// AbrirPipes crea/abre los FIFOs necesarios para la comunicación half-duplex.
// En Linux se requieren dos pipes: uno para peticiones y otro para respuestas.
// Si no existen, los crea con mkfifo.
func AbrirPipes(pipePeticiones, pipeRespuestas string) (*os.File, *os.File, error) {
	// Crear FIFOs si no existen
	if err := crearFIFO(pipePeticiones); err != nil {
		return nil, nil, fmt.Errorf("AbrirPipes: %w", err)
	}
	if err := crearFIFO(pipeRespuestas); err != nil {
		return nil, nil, fmt.Errorf("AbrirPipes: %w", err)
	}

	// Abrir pipe de lectura (bloqueante hasta que alguien escriba)
	lectura, err := os.OpenFile(pipePeticiones, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, nil, fmt.Errorf("AbrirPipes: no se pudo abrir pipe de lectura: %w", err)
	}

	// Abrir pipe de escritura
	escritura, err := os.OpenFile(pipeRespuestas, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		lectura.Close()
		return nil, nil, fmt.Errorf("AbrirPipes: no se pudo abrir pipe de escritura: %w", err)
	}

	return lectura, escritura, nil
}

// crearFIFO crea un FIFO (named pipe) si no existe.
func crearFIFO(ruta string) error {
	if _, err := os.Stat(ruta); os.IsNotExist(err) {
		// Crear FIFO con permisos 0666 (serán restringidos por umask)
		if err := syscall.Mkfifo(ruta, 0666); err != nil {
			return fmt.Errorf("crearFIFO: %w", err)
		}
	}
	return nil
}
