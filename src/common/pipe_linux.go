//go:build linux

// Package common proporciona utilidades compartidas por todos los servicios.
// BORRADOR: implementación temporal para permitir compilación de gesprog.
package common

import (
	"fmt"
	"os"
	"syscall"
)

// AbrirPipes abre los dos FIFOs necesarios para comunicación half-duplex en Linux.
// Crea los FIFOs si no existen.
func AbrirPipes(pipePeticiones, pipeRespuestas string) (*os.File, *os.File, error) {
	if err := crearFIFO(pipePeticiones); err != nil {
		return nil, nil, err
	}
	if err := crearFIFO(pipeRespuestas); err != nil {
		return nil, nil, err
	}
	entrada, err := os.OpenFile(pipePeticiones, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, nil, fmt.Errorf("AbrirPipes: %w", err)
	}
	salida, err := os.OpenFile(pipeRespuestas, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		entrada.Close()
		return nil, nil, fmt.Errorf("AbrirPipes: %w", err)
	}
	return entrada, salida, nil
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
