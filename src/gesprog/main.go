package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

// main es el punto de entrada del servicio gesprog.
func main() {
	// --- flags de línea de comandos ---
	pipePeticiones := flag.String("p", "", "tubería nombrada para recibir peticiones (obligatorio)")
	pipeRespuestas := flag.String("c", "", "tubería nombrada para enviar respuestas (solo Linux, half-duplex)")
	aralmac := flag.String("x", "", "ruta del directorio aralmac (obligatorio)")
	flag.Parse()

	// Validar flags obligatorios
	if *pipePeticiones == "" {
		fmt.Fprintln(os.Stderr, "error: se requiere -p <tuberia-nombrada>")
		flag.Usage()
		os.Exit(1)
	}
	if *aralmac == "" {
		fmt.Fprintln(os.Stderr, "error: se requiere -x <info-aralmac>")
		flag.Usage()
		os.Exit(1)
	}

	// En Linux se usan dos pipes (half-duplex).
	// En Windows el pipe es full-duplex y -c es opcional.
	pipeSalida := *pipeRespuestas
	if pipeSalida == "" {
		pipeSalida = *pipePeticiones
	}

	// Inicializar almacenamiento
	if err := InicializarAlmacenamiento(*aralmac); err != nil {
		log.Fatalf("gesprog: error inicializando aralmac: %v", err)
	}

	log.Printf("gesprog: iniciando en pipe=%s aralmac=%s", *pipePeticiones, *aralmac)

	// Arrancar servidor (bloqueante hasta recibir Terminar)
	if err := Servidor(*pipePeticiones, pipeSalida); err != nil {
		log.Fatalf("gesprog: error en servidor: %v", err)
	}

	log.Println("gesprog: terminado correctamente")
}
