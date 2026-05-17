package gesprog

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/SofiAlfonso/Ejecutor_de_Lotes/src/common"
)

// TestServidor_GuardarYLeer prueba el flujo completo del servidor:
// - Crea pipes temporales
// - Lanza Servidor en una goroutine
// - Envía una petición Guardar
// - Lee la respuesta y verifica que contiene id-programa
func TestServidor_GuardarYLeer(t *testing.T) {
	// 1. Preparar entorno de almacenamiento temporal
	tempDir := t.TempDir()
	err := InicializarAlmacenamiento(tempDir)
	if err != nil {
		t.Fatalf("InicializarAlmacenamiento: %v", err)
	}

	// 2. Crear rutas para pipes temporales (diferentes según SO)
	//    Usamos rutas dentro del directorio temporal para no ensuciar el sistema.
	pipePeticiones := filepath.Join(tempDir, "peticiones")
	pipeRespuestas := filepath.Join(tempDir, "respuestas")
	if isWindows() {
		// En Windows los pipes nombrados tienen formato \\.\pipe\nombre
		// pero el paquete common ya maneja eso. Para pruebas, podemos usar
		// un nombre único dentro del espacio de pipes.
		pipePeticiones = `\\.\pipe\gesprog_test_peticiones`
		pipeRespuestas = `\\.\pipe\gesprog_test_respuestas`
		// En Windows con full-duplex, `AbrirPipes` ignora pipeRespuestas,
		// pero lo pasamos igual.
	} else {
		// Linux: creamos los FIFOs manualmente (el código common los crea, pero mejor asegurar)
		// No es necesario, AbrirPipes los creará con mkfifo.
	}

	// 3. Ejecutar Servidor en una goroutine
	errc := make(chan error, 1)
	go func() {
		err := Servidor(pipePeticiones, pipeRespuestas)
		errc <- err
	}()

	// Dar tiempo para que el servidor abra los pipes y quede escuchando
	time.Sleep(500 * time.Millisecond)

	// 4. Crear un cliente que se conecte a los pipes y envíe una petición
	//    Usamos las mismas funciones de common para abrir los pipes.
	//    Como `common.AbrirPipes` espera dos parámetros, los usamos igual.
	entrada, salida, err := common.AbrirPipes(pipePeticiones, pipeRespuestas)
	if err != nil {
		t.Fatalf("no se pudo abrir pipes para el cliente: %v", err)
	}
	defer entrada.Close()
	defer salida.Close()

	// 5. Crear un ejecutable temporal para guardar
	ejecutable := crearEjecutableTemporal(t, "test_prog")

	// 6. Construir petición Guardar
	req := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Guardar",
		Ejecutable: ejecutable,
		Args:       []string{"-test"},
		Env:        []string{"FOO=BAR"},
	}
	reqJSON, _ := json.Marshal(req)
	// Añadir nueva línea (protocolo)
	reqJSON = append(reqJSON, '\n')

	// 7. Escribir la petición en el pipe de entrada del servidor
	if _, err := entrada.Write(reqJSON); err != nil {
		t.Fatalf("error escribiendo petición: %v", err)
	}

	// 8. Leer la respuesta (el servidor escribe en `salida`)
	buf := make([]byte, 4096)
	n, err := salida.Read(buf)
	if err != nil {
		t.Fatalf("error leyendo respuesta: %v", err)
	}
	respuesta := buf[:n]

	// 9. Parsear respuesta
	var resp Respuesta
	if err := json.Unmarshal(respuesta, &resp); err != nil {
		t.Fatalf("no se pudo parsear respuesta: %v, texto: %s", err, respuesta)
	}
	if resp.Estado != "ok" {
		t.Fatalf("Guardar falló: %s", resp.Mensaje)
	}
	if resp.IDPrograma == "" {
		t.Error("ID de programa vacío")
	}

	// 10. Terminar el servidor enviando Terminar
	reqTerm := Peticion{
		Servicio:  "gesprog",
		Operacion: "Terminar",
	}
	termJSON, _ := json.Marshal(reqTerm)
	termJSON = append(termJSON, '\n')
	entrada.Write(termJSON)

	// 11. Esperar a que el servidor termine (sin error)
	select {
	case err := <-errc:
		if err != nil {
			t.Errorf("servidor terminó con error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("el servidor no terminó en tiempo esperado")
	}
}

// TestServidor_OperacionDesconocida prueba que el servidor responde error.
func TestServidor_OperacionDesconocida(t *testing.T) {
	tempDir := t.TempDir()
	InicializarAlmacenamiento(tempDir)

	pipePeticiones := filepath.Join(tempDir, "peticiones2")
	pipeRespuestas := filepath.Join(tempDir, "respuestas2")
	if isWindows() {
		pipePeticiones = `\\.\pipe\gesprog_test2_peticiones`
		pipeRespuestas = `\\.\pipe\gesprog_test2_respuestas`
	}

	errc := make(chan error, 1)
	go func() {
		errc <- Servidor(pipePeticiones, pipeRespuestas)
	}()
	time.Sleep(500 * time.Millisecond)

	entrada, salida, err := common.AbrirPipes(pipePeticiones, pipeRespuestas)
	if err != nil {
		t.Fatalf("no se pudo abrir pipes: %v", err)
	}
	defer entrada.Close()
	defer salida.Close()

	req := []byte(`{"servicio":"gesprog","operacion":"Volar"}\n`)
	entrada.Write(req)

	buf := make([]byte, 4096)
	n, _ := salida.Read(buf)
	var resp Respuesta
	json.Unmarshal(buf[:n], &resp)
	if resp.Estado != "error" || resp.Mensaje != "operacion desconocida" {
		t.Errorf("respuesta incorrecta: %+v", resp)
	}

	// Terminar
	entrada.Write([]byte(`{"servicio":"gesprog","operacion":"Terminar"}\n`))
	<-errc
}

// isWindows detecta si estamos en Windows (para usar rutas adecuadas en pruebas)
func isWindows() bool {
	return os.PathSeparator == '\\'
}
