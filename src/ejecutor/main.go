// Package main implementa el servicio ejecutor de procesos de lotes.
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// main es el punto de entrada del servicio ejecutor.
func main() {
	// --- flags de línea de comandos ---
	pipePeticiones := flag.String("e", "", "tubería nombrada para recibir peticiones (obligatorio)")
	pipeRespuestas := flag.String("d", "", "tubería nombrada para enviar respuestas (solo Linux, half-duplex)")
	aralmac := flag.String("x", "", "ruta del directorio aralmac (obligatorio)")
	flag.Parse()

	// Validar flags obligatorios
	if *pipePeticiones == "" {
		fmt.Fprintln(os.Stderr, "error: se requiere -e <tuberia-nombrada>")
		flag.Usage()
		os.Exit(1)
	}
	if *aralmac == "" {
		fmt.Fprintln(os.Stderr, "error: se requiere -x <info-aralmac>")
		flag.Usage()
		os.Exit(1)
	}

	// En Linux se usan dos pipes (half-duplex).
	// En Windows el pipe es full-duplex y -d es opcional.
	pipeSalida := *pipeRespuestas
	if pipeSalida == "" {
		pipeSalida = *pipePeticiones
	}

	// Inicializar almacenamiento
	if err := InicializarAlmacenamiento(*aralmac); err != nil {
		log.Fatalf("ejecutor: error inicializando aralmac: %v", err)
	}

	log.Printf("ejecutor: iniciando en pipe=%s aralmac=%s", *pipePeticiones, *aralmac)

	// Arrancar servidor (bloqueante hasta que el servicio pase a Terminado)
	if err := Servidor(*pipePeticiones, pipeSalida); err != nil {
		log.Fatalf("ejecutor: error en servidor: %v", err)
	}

	log.Println("ejecutor: terminado correctamente")
}
