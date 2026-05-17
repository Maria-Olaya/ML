// ejecutor_test.go
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ---------------------------------------------------------------------
// Pruebas de la máquina de estados del servicio
// ---------------------------------------------------------------------

func resetServicio() {
	servicio.mu.Lock()
	defer servicio.mu.Unlock()
	servicio.current = estadoEjecutar
}

func TestServicioEstados(t *testing.T) {
	resetServicio()

	if !ServicioAceptaEjecuciones() {
		t.Error("debería aceptar ejecuciones al inicio")
	}
	if !ServicioAceptaPeticiones() {
		t.Error("debería aceptar otras peticiones al inicio")
	}

	err := SuspenderServicio()
	if err != nil {
		t.Fatal(err)
	}
	if ServicioAceptaEjecuciones() {
		t.Error("no debe aceptar ejecuciones cuando está suspendido")
	}
	if !ServicioAceptaPeticiones() {
		t.Error("debe aceptar otras peticiones cuando está suspendido")
	}

	err = ReanudarServicio()
	if err != nil {
		t.Fatal(err)
	}
	if !ServicioAceptaEjecuciones() {
		t.Error("debe aceptar ejecuciones después de reanudar")
	}

	err = PararServicio()
	if err != nil {
		t.Fatal(err)
	}
	if ServicioAceptaEjecuciones() {
		t.Error("no debe aceptar ejecuciones en estado Parando")
	}
	if !ServicioAceptaPeticiones() {
		t.Error("debe aceptar otras peticiones en Parando")
	}
}

func TestTransicionesInvalidas(t *testing.T) {
	resetServicio()

	_ = SuspenderServicio()
	err := SuspenderServicio()
	if err == nil {
		t.Error("suspender dos veces debe fallar")
	}

	resetServicio()
	err = ReanudarServicio()
	if err == nil {
		t.Error("reanudar sin estar suspendido debe fallar")
	}

	resetServicio()
	_ = SuspenderServicio()
	err = PararServicio()
	if err == nil {
		t.Error("parar desde suspendido debe fallar")
	}
}

// ---------------------------------------------------------------------
// Pruebas de registro de procesos
// ---------------------------------------------------------------------

func limpiarRegistroProcesos() {
	muProcesos.Lock()
	defer muProcesos.Unlock()
	registroProcesos = make(map[string]*InfoProceso)
	contadorActivos = 0
}

func TestRegistroProcesos(t *testing.T) {
	limpiarRegistroProcesos()

	info := RegistrarProceso("e-001", "p-001")
	if info == nil {
		t.Fatal("RegistrarProceso devolvió nil")
	}
	if info.IDEjecucion != "e-001" || info.IDPrograma != "p-001" {
		t.Errorf("datos incorrectos: %+v", info)
	}
	if info.Estado != ProcesoEjecutando {
		t.Errorf("estado inicial debe ser Ejecutando, got %v", info.Estado)
	}

	obtenido, err := ObtenerProceso("e-001")
	if err != nil {
		t.Fatal(err)
	}
	if obtenido != info {
		t.Error("ObtenerProceso no devolvió la misma referencia")
	}

	_, err = ObtenerProceso("e-999")
	if err == nil {
		t.Error("debe fallar para ID inexistente")
	}
}

func TestMarcarTerminado(t *testing.T) {
	limpiarRegistroProcesos()
	resetServicio()

	info := RegistrarProceso("e-001", "p-001")
	MarcarTerminado("e-001", 42)

	if info.Estado != ProcesoTerminado {
		t.Errorf("estado debe ser Terminado, got %v", info.Estado)
	}
	if info.CodigoSalida != 42 {
		t.Errorf("codigoSalida debe ser 42, got %d", info.CodigoSalida)
	}
	if !info.Terminado {
		t.Error("Terminado debe ser true")
	}

	muProcesos.RLock()
	activos := contadorActivos
	muProcesos.RUnlock()
	if activos != 0 {
		t.Errorf("contadorActivos debe ser 0, got %d", activos)
	}
}

func TestAutoTerminarEnParando(t *testing.T) {
	limpiarRegistroProcesos()
	resetServicio()

	RegistrarProceso("e-001", "p-001")
	_ = PararServicio()
	MarcarTerminado("e-001", 0)

	time.Sleep(10 * time.Millisecond)
	if !ServicioEstaTerminado() {
		t.Error("el servicio debería terminar automáticamente")
	}
}

func TestListarProcesos(t *testing.T) {
	limpiarRegistroProcesos()
	RegistrarProceso("e-001", "p-001")
	RegistrarProceso("e-002", "p-002")

	lista := ListarProcesos()
	if len(lista) != 2 {
		t.Fatalf("se esperaban 2 procesos, got %d", len(lista))
	}
}

// ---------------------------------------------------------------------
// Pruebas de almacenamiento (persistencia)
// ---------------------------------------------------------------------

func setupAlmacenamiento(t *testing.T) func() {
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "programas"), 0755)
	os.MkdirAll(filepath.Join(tempDir, "ficheros"), 0755)
	err := InicializarAlmacenamiento(tempDir)
	if err != nil {
		t.Fatalf("InicializarAlmacenamiento: %v", err)
	}
	return func() {}
}

func TestVerificarPrograma(t *testing.T) {
	cleanup := setupAlmacenamiento(t)
	defer cleanup()

	// Crear un programa falso
	meta := metadataPrograma{
		ID:     "p-001",
		Nombre: "test",
		Args:   []string{"arg1"},
		Env:    []string{"A=B"},
	}
	metaPath := filepath.Join(rutaAralmac, "programas", "p-001.json")
	binPath := filepath.Join(rutaAralmac, "programas", "p-001.bin")
	data, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(metaPath, data, 0644)
	os.WriteFile(binPath, []byte("fake binary"), 0755)

	rutaBin, meta2, err := VerificarPrograma("p-001")
	if err != nil {
		t.Fatal(err)
	}
	if rutaBin != binPath {
		t.Error("ruta del binario incorrecta")
	}
	if meta2.Nombre != "test" {
		t.Error("metadatos incorrectos")
	}
}

func TestVerificarFichero(t *testing.T) {
	cleanup := setupAlmacenamiento(t)
	defer cleanup()

	ficheroPath := filepath.Join(rutaAralmac, "ficheros", "f-001.dat")
	os.WriteFile(ficheroPath, []byte("dummy"), 0644)

	ruta, err := VerificarFichero("f-001")
	if err != nil {
		t.Fatal(err)
	}
	if ruta != ficheroPath {
		t.Error("ruta incorrecta")
	}

	_, err = VerificarFichero("f-999")
	if err == nil {
		t.Error("debe fallar para fichero inexistente")
	}
}

func TestGuardarEjecucion(t *testing.T) {
	cleanup := setupAlmacenamiento(t)
	defer cleanup()

	info := &InfoProceso{
		IDEjecucion:  "e-001",
		IDPrograma:   "p-001",
		Estado:       ProcesoEjecutando,
		CodigoSalida: 0,
		Terminado:    false,
	}
	err := GuardarEjecucion(info)
	if err != nil {
		t.Fatal(err)
	}
	ruta := filepath.Join(rutaAralmac, "ejecuciones", "e-001.json")
	if _, err := os.Stat(ruta); os.IsNotExist(err) {
		t.Error("archivo no creado")
	}
}

// ---------------------------------------------------------------------
// Pruebas de LanzarProceso (solo un proceso, sin forzar IDs únicos)
// ---------------------------------------------------------------------

func TestLanzarProceso(t *testing.T) {
	cleanup := setupAlmacenamiento(t)
	defer cleanup()

	// Crear un programa simulado (un script simple)
	// Para portabilidad, usamos un comando que existe en ambos sistemas.
	// En Linux: /bin/echo; en Windows: cmd.exe (sin argumentos, solo sale)
	// Pero no necesitamos que haga nada, solo que exista.
	ejecutable := ""
	if isWindows() {
		ejecutable = "cmd.exe"
	} else {
		ejecutable = "/bin/echo"
	}
	meta := metadataPrograma{
		ID:     "p-001",
		Nombre: filepath.Base(ejecutable),
		Args:   []string{},
		Env:    nil,
	}
	metaPath := filepath.Join(rutaAralmac, "programas", "p-001.json")
	binPath := filepath.Join(rutaAralmac, "programas", "p-001.bin")
	data, _ := json.MarshalIndent(meta, "", "  ")
	os.WriteFile(metaPath, data, 0644)
	os.WriteFile(binPath, []byte("dummy"), 0755) // en Windows no se usa realmente

	id, err := LanzarProceso("p-001", "", "", "")
	if err != nil {
		t.Fatalf("LanzarProceso falló: %v", err)
	}
	// Como el generador siempre devuelve "e-0001", verificamos que el ID sea ese.
	if id != "e-0001" {
		t.Errorf("ID esperado e-0001, got %s", id)
	}
	// Esperar un poco a que termine
	time.Sleep(100 * time.Millisecond)
	info, err := ObtenerProceso(id)
	if err != nil {
		t.Fatal(err)
	}
	info.mu.RLock()
	terminado := info.Terminado
	codigo := info.CodigoSalida
	info.mu.RUnlock()
	if !terminado {
		t.Error("el proceso debería haber terminado")
	}
	if codigo != 0 {
		t.Errorf("código de salida debe ser 0, got %d", codigo)
	}
}

// ---------------------------------------------------------------------
// Pruebas de operaciones (despachador, sin ejecución real)
// ---------------------------------------------------------------------

func TestProcesarPeticion_EjecutarError(t *testing.T) {
	cleanup := setupAlmacenamiento(t)
	defer cleanup()

	// No creamos el programa, así que debe fallar con "programa no encontrado"
	req := Peticion{
		Operacion:  "Ejecutar",
		IDPrograma: "p-999",
	}
	raw, _ := json.Marshal(req)
	respBytes := ProcesarPeticion(raw)
	var resp Respuesta
	json.Unmarshal(respBytes, &resp)
	if resp.Estado != "error" || resp.Mensaje != "programa no encontrado" {
		t.Errorf("respuesta incorrecta: %+v", resp)
	}
}

func TestProcesarPeticion_EstadoVacio(t *testing.T) {
	req := Peticion{Operacion: "Estado"}
	raw, _ := json.Marshal(req)
	respBytes := ProcesarPeticion(raw)
	var resp Respuesta
	json.Unmarshal(respBytes, &resp)
	if resp.Estado != "ok" {
		t.Error("Estado debería ser ok incluso con lista vacía")
	}
	if len(resp.Procesos) != 0 {
		t.Error("debería devolver lista vacía")
	}
}

func TestProcesarPeticion_OperacionDesconocida(t *testing.T) {
	req := Peticion{Operacion: "Volar"}
	raw, _ := json.Marshal(req)
	respBytes := ProcesarPeticion(raw)
	var resp Respuesta
	json.Unmarshal(respBytes, &resp)
	if resp.Estado != "error" || resp.Mensaje != "operacion desconocida" {
		t.Errorf("respuesta incorrecta: %+v", resp)
	}
}

func TestProcesarPeticion_Suspender(t *testing.T) {
	resetServicio()
	req := Peticion{Operacion: "Suspender"}
	raw, _ := json.Marshal(req)
	respBytes := ProcesarPeticion(raw)
	var resp Respuesta
	json.Unmarshal(respBytes, &resp)
	if resp.Estado != "ok" {
		t.Fatalf("Suspender falló: %s", resp.Mensaje)
	}
	if !ServicioAceptaPeticiones() || ServicioAceptaEjecuciones() {
		t.Error("servicio debería estar suspendido")
	}
}

func TestProcesarPeticion_Reasumir(t *testing.T) {
	resetServicio()
	_ = SuspenderServicio()
	req := Peticion{Operacion: "Reasumir"}
	raw, _ := json.Marshal(req)
	respBytes := ProcesarPeticion(raw)
	var resp Respuesta
	json.Unmarshal(respBytes, &resp)
	if resp.Estado != "ok" {
		t.Fatalf("Reasumir falló: %s", resp.Mensaje)
	}
	if !ServicioAceptaEjecuciones() {
		t.Error("servicio debería estar activo")
	}
}

func TestProcesarPeticion_Parar(t *testing.T) {
	resetServicio()
	req := Peticion{Operacion: "Parar"}
	raw, _ := json.Marshal(req)
	respBytes := ProcesarPeticion(raw)
	var resp Respuesta
	json.Unmarshal(respBytes, &resp)
	if resp.Estado != "ok" {
		t.Fatalf("Parar falló: %s", resp.Mensaje)
	}
	// El servicio debe estar en Parando
	if ServicioAceptaEjecuciones() {
		t.Error("no debe aceptar ejecuciones después de Parar")
	}
	if !ServicioAceptaPeticiones() {
		t.Error("debe aceptar otras peticiones")
	}
}

// Helper para detectar Windows
func isWindows() bool {
	return os.PathSeparator == '\\'
}
