// Package gesprog gestiona programas almacenados en disco.
package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
)

// Servidor escucha peticiones desde el pipe de entrada y escribe respuestas
// en el pipe de salida. Corre hasta que el servicio pase a estado Terminado.
func Servidor(pipeLectura, pipeEscritura string) error {
	// Abrir pipe de lectura (bloqueante hasta que haya un escritor)
	entrada, err := abrirPipeLectura(pipeLectura)
	if err != nil {
		return fmt.Errorf("servidor: no se pudo abrir pipe de lectura: %w", err)
	}
	defer entrada.Close()

	// Abrir pipe de escritura
	salida, err := abrirPipeEscritura(pipeEscritura)
	if err != nil {
		return fmt.Errorf("servidor: no se pudo abrir pipe de escritura: %w", err)
	}
	defer salida.Close()

	scanner := bufio.NewScanner(entrada)
	writer := bufio.NewWriter(salida)

	for scanner.Scan() {
		linea := scanner.Bytes()
		if len(linea) == 0 {
			continue
		}

		// Procesar petición y obtener respuesta
		respuesta := ProcesarPeticion(linea)

		// Escribir respuesta seguida de salto de línea (protocolo)
		if _, err := writer.Write(respuesta); err != nil {
			return fmt.Errorf("servidor: error escribiendo respuesta: %w", err)
		}
		if err := writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("servidor: error escribiendo newline: %w", err)
		}
		if err := writer.Flush(); err != nil {
			return fmt.Errorf("servidor: error en flush: %w", err)
		}

		// Si la operación fue Terminar, salir del bucle
		if EstaTerminado() {
			break
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("servidor: error leyendo pipe: %w", err)
	}
	return nil
}

// abrirPipeLectura abre el pipe de entrada en modo lectura.
// En Linux es un FIFO, en Windows es un Named Pipe.
func abrirPipeLectura(ruta string) (*os.File, error) {
	f, err := os.OpenFile(ruta, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("abrirPipeLectura: %w", err)
	}
	return f, nil
}

// abrirPipeEscritura abre el pipe de salida en modo escritura.
// En Linux es un FIFO, en Windows es un Named Pipe.
func abrirPipeEscritura(ruta string) (*os.File, error) {
	f, err := os.OpenFile(ruta, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		return nil, fmt.Errorf("abrirPipeEscritura: %w", err)
	}
	return f, nil
}
