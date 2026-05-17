// Package main implementa el servicio ejecutor de procesos de lotes.
package main

import (
	"bufio"
	"fmt"
	"io"
	"sync"

	"github.com/SofiAlfonso/Ejecutor_de_Lotes/src/common"
)

// maxMsgLen es el tamaño máximo de mensaje definido en el protocolo.
const maxMsgLen = 4096

// Servidor escucha peticiones desde el pipe de entrada y escribe respuestas
// en el pipe de salida. Lanza una goroutine por petición para no bloquear
// ejecuciones paralelas. Corre hasta que el servicio pase a estado Terminado.
func Servidor(pipePeticiones, pipeRespuestas string) error {
	entrada, salida, err := common.AbrirPipes(pipePeticiones, pipeRespuestas)
	if err != nil {
		return fmt.Errorf("servidor: %w", err)
	}
	defer entrada.Close()
	defer salida.Close()

	scanner := bufio.NewScanner(entrada)
	scanner.Buffer(make([]byte, maxMsgLen), maxMsgLen)

	var wg sync.WaitGroup
	var mu sync.Mutex
	writer := bufio.NewWriter(salida)

	for scanner.Scan() {
		linea := scanner.Bytes()
		if len(linea) == 0 {
			continue
		}

		// Copiar la línea para pasarla a la goroutine de forma segura
		copia := make([]byte, len(linea))
		copy(copia, linea)

		wg.Add(1)
		go func(peticion []byte) {
			defer wg.Done()
			respuesta := ProcesarPeticion(peticion)

			mu.Lock()
			defer mu.Unlock()
			_, _ = writer.Write(respuesta)
			_ = writer.WriteByte('\n')
			_ = writer.Flush()
		}(copia)

		if ServicioEstaTerminado() {
			break
		}
	}

	// Esperar a que todas las goroutines terminen de escribir
	// antes de cerrar los pipes.
	wg.Wait()

	if err := scanner.Err(); err != nil && err != io.EOF {
		return fmt.Errorf("servidor: error leyendo pipe: %w", err)
	}
	return nil
}
