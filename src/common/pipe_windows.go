//go:build windows

package common

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows"
)

// AbrirPipes crea un named pipe full-duplex en Windows y espera a que un cliente se conecte.
// - pipePeticiones: nombre del pipe (ej: `\\.\pipe\gesprog_pipe`)
// - pipeRespuestas: se ignora (full-duplex)
// Retorna dos *os.File que apuntan al mismo pipe (lectura y escritura).
func AbrirPipes(pipePeticiones, pipeRespuestas string) (*os.File, *os.File, error) {
	// Convertir nombre de pipe a UTF-16
	pipeName, err := windows.UTF16PtrFromString(pipePeticiones)
	if err != nil {
		return nil, nil, fmt.Errorf("AbrirPipes: nombre de pipe inválido: %w", err)
	}

	// Crear el named pipe como servidor
	handle, err := windows.CreateNamedPipe(
		pipeName,
		windows.PIPE_ACCESS_DUPLEX,
		windows.PIPE_TYPE_MESSAGE|windows.PIPE_READMODE_MESSAGE|windows.PIPE_WAIT,
		1,    // max instances
		4096, // output buffer size
		4096, // input buffer size
		0,    // default timeout
		nil,  // security attributes
	)
	if err != nil {
		return nil, nil, fmt.Errorf("AbrirPipes: CreateNamedPipe falló: %w", err)
	}

	// Esperar a que un cliente se conecte (bloqueante)
	if err := windows.ConnectNamedPipe(handle, nil); err != nil && err != windows.ERROR_PIPE_CONNECTED {
		windows.CloseHandle(handle)
		return nil, nil, fmt.Errorf("AbrirPipes: ConnectNamedPipe falló: %w", err)
	}

	// Convertir el handle a *os.File para usar con bufio
	file := os.NewFile(uintptr(handle), pipePeticiones)
	return file, file, nil
}
