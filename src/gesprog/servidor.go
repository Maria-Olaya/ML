package main

import (
	"bufio"
	"fmt"
	"io"

	"github.com/SofiAlfonso/Ejecutor_de_Lotes/src/common"
)

const maxMsgLen = 4096

// Servidor escucha peticiones desde el pipe de entrada y escribe respuestas
// en el pipe de salida. Corre hasta que el servicio pase a estado Terminado.
func Servidor(pipePeticiones, pipeRespuestas string) error {
	// Usar common.AbrirPipes: en Windows crea el pipe y espera conexión;
	// en Linux abre los FIFOs existentes.
	entrada, salida, err := common.AbrirPipes(pipePeticiones, pipeRespuestas)
	if err != nil {
		return fmt.Errorf("servidor: %w", err)
	}
	defer entrada.Close()
	defer salida.Close()

	scanner := bufio.NewScanner(entrada)
	scanner.Buffer(make([]byte, maxMsgLen), maxMsgLen)
	writer := bufio.NewWriter(salida)

	for scanner.Scan() {
		linea := scanner.Bytes()
		if len(linea) == 0 {
			continue
		}

		respuesta := ProcesarPeticion(linea)

		if _, err := writer.Write(respuesta); err != nil {
			return fmt.Errorf("servidor: error escribiendo respuesta: %w", err)
		}
		if err := writer.WriteByte('\n'); err != nil {
			return fmt.Errorf("servidor: error escribiendo newline: %w", err)
		}
		if err := writer.Flush(); err != nil {
			return fmt.Errorf("servidor: error en flush: %w", err)
		}

		if EstaTerminado() {
			break
		}
	}

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("servidor: error leyendo pipe: %w", err)
	}
	return nil
}
