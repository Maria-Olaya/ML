package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// setupOperaciones inicializa el almacenamiento en un directorio temporal
// y resetea el estado del servicio a Corriendo.
func setupOperaciones(t *testing.T) func() {
	tempDir := t.TempDir()
	err := InicializarAlmacenamiento(tempDir)
	if err != nil {
		t.Fatalf("InicializarAlmacenamiento falló: %v", err)
	}
	// Asegurar estado inicial (por si pruebas anteriores lo cambiaron)
	_ = Reanudar()
	// Nota: el generador de IDs es placeholder, siempre devuelve "p-0001".
	// Por eso no se pueden guardar dos programas diferentes en la misma ejecución.
	// Cada test usa su propio TempDir, así que no hay conflicto.
	return func() {
		// cleanup implícito: TempDir se borra automáticamente
	}
}

// crearEjecutableTemporal crea un archivo ejecutable simulado.
func crearEjecutableTemporal(t *testing.T, nombre string) string {
	dir := t.TempDir()
	ruta := filepath.Join(dir, nombre)
	contenido := []byte("#!/bin/bash\necho 'hola mundo'")
	err := os.WriteFile(ruta, contenido, 0755)
	if err != nil {
		t.Fatalf("no se pudo crear ejecutable temporal: %v", err)
	}
	return ruta
}

// TestProcesarPeticion_GuardarOk prueba el guardado exitoso.
func TestProcesarPeticion_GuardarOk(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	ejecutable := crearEjecutableTemporal(t, "mi_prog")
	req := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Guardar",
		Ejecutable: ejecutable,
		Args:       []string{"-a", "-b"},
		Env:        []string{"X=1", "Y=2"},
	}
	reqJSON, _ := json.Marshal(req)
	respBytes := ProcesarPeticion(reqJSON)

	var resp Respuesta
	err := json.Unmarshal(respBytes, &resp)
	if err != nil {
		t.Fatalf("no se pudo parsear respuesta: %v", err)
	}
	if resp.Estado != "ok" {
		t.Fatalf("Guardar falló: %s", resp.Mensaje)
	}
	if resp.IDPrograma == "" {
		t.Error("IDPrograma vacío")
	}
}

// TestProcesarPeticion_GuardarErrorEjecutableFaltante prueba el campo obligatorio.
func TestProcesarPeticion_GuardarErrorEjecutableFaltante(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	req := Peticion{
		Servicio:  "gesprog",
		Operacion: "Guardar",
		// Ejecutable ausente
	}
	reqJSON, _ := json.Marshal(req)
	respBytes := ProcesarPeticion(reqJSON)

	var resp Respuesta
	json.Unmarshal(respBytes, &resp)
	if resp.Estado != "error" || resp.Mensaje != "falta campo: ejecutable" {
		t.Errorf("mensaje de error incorrecto: %+v", resp)
	}
}

// TestProcesarPeticion_LeerPorIDOk prueba lectura de un programa específico.
func TestProcesarPeticion_LeerPorIDOk(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	// 1. Guardar un programa
	ejecutable := crearEjecutableTemporal(t, "leer_prog")
	guardarReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Guardar",
		Ejecutable: ejecutable,
		Args:       []string{"-x"},
	}
	guardarJSON, _ := json.Marshal(guardarReq)
	respGuardar := ProcesarPeticion(guardarJSON)
	var guardarResp Respuesta
	json.Unmarshal(respGuardar, &guardarResp)
	id := guardarResp.IDPrograma

	// 2. Leer por ID
	leerReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Leer",
		IDPrograma: id,
	}
	leerJSON, _ := json.Marshal(leerReq)
	respLeer := ProcesarPeticion(leerJSON)
	var leerResp Respuesta
	json.Unmarshal(respLeer, &leerResp)

	if leerResp.Estado != "ok" {
		t.Fatalf("Leer falló: %s", leerResp.Mensaje)
	}
	if leerResp.Programa == nil || leerResp.Programa.ID != id {
		t.Errorf("programa incorrecto: %+v", leerResp.Programa)
	}
}

// TestProcesarPeticion_LeerTodos prueba listar todos los IDs.
func TestProcesarPeticion_LeerTodos(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	// Guardar un programa (solo cabe uno por el placeholder de ID)
	ejecutable := crearEjecutableTemporal(t, "listar_prog")
	guardarReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Guardar",
		Ejecutable: ejecutable,
	}
	guardarJSON, _ := json.Marshal(guardarReq)
	ProcesarPeticion(guardarJSON)

	// Leer todos
	leerReq := Peticion{
		Servicio:  "gesprog",
		Operacion: "Leer",
		// sin IDPrograma
	}
	leerJSON, _ := json.Marshal(leerReq)
	respBytes := ProcesarPeticion(leerJSON)
	var resp Respuesta
	json.Unmarshal(respBytes, &resp)

	if resp.Estado != "ok" {
		t.Fatalf("Leer todos falló: %s", resp.Mensaje)
	}
	if len(resp.Programas) != 1 || resp.Programas[0] != "p-0001" {
		t.Errorf("lista incorrecta: %v", resp.Programas)
	}
}

// TestProcesarPeticion_ActualizarOk prueba actualizar la ruta del ejecutable.
func TestProcesarPeticion_ActualizarOk(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	ejecutable1 := crearEjecutableTemporal(t, "v1")
	guardarReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Guardar",
		Ejecutable: ejecutable1,
	}
	guardarJSON, _ := json.Marshal(guardarReq)
	respGuardar := ProcesarPeticion(guardarJSON)
	var guardarResp Respuesta
	json.Unmarshal(respGuardar, &guardarResp)
	id := guardarResp.IDPrograma

	ejecutable2 := crearEjecutableTemporal(t, "v2")
	actualizarReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Actualizar",
		IDPrograma: id,
		Ruta:       ejecutable2,
	}
	actJSON, _ := json.Marshal(actualizarReq)
	respAct := ProcesarPeticion(actJSON)
	var actResp Respuesta
	json.Unmarshal(respAct, &actResp)

	if actResp.Estado != "ok" {
		t.Fatalf("Actualizar falló: %s", actResp.Mensaje)
	}

	// Verificar que el nombre cambió
	leerReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Leer",
		IDPrograma: id,
	}
	leerJSON, _ := json.Marshal(leerReq)
	respLeer := ProcesarPeticion(leerJSON)
	var leerResp Respuesta
	json.Unmarshal(respLeer, &leerResp)
	if leerResp.Programa.Nombre != "v2" {
		t.Errorf("nombre no actualizado: %s", leerResp.Programa.Nombre)
	}
}

// TestProcesarPeticion_BorrarOk prueba eliminación.
func TestProcesarPeticion_BorrarOk(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	ejecutable := crearEjecutableTemporal(t, "borrar_prog")
	guardarReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Guardar",
		Ejecutable: ejecutable,
	}
	guardarJSON, _ := json.Marshal(guardarReq)
	respGuardar := ProcesarPeticion(guardarJSON)
	var guardarResp Respuesta
	json.Unmarshal(respGuardar, &guardarResp)
	id := guardarResp.IDPrograma

	borrarReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Borrar",
		IDPrograma: id,
	}
	borrarJSON, _ := json.Marshal(borrarReq)
	respBorrar := ProcesarPeticion(borrarJSON)
	var borrarResp Respuesta
	json.Unmarshal(respBorrar, &borrarResp)
	if borrarResp.Estado != "ok" {
		t.Fatalf("Borrar falló: %s", borrarResp.Mensaje)
	}

	// Verificar que ya no existe
	leerReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Leer",
		IDPrograma: id,
	}
	leerJSON, _ := json.Marshal(leerReq)
	respLeer := ProcesarPeticion(leerJSON)
	var leerResp Respuesta
	json.Unmarshal(respLeer, &leerResp)
	if leerResp.Estado != "error" {
		t.Error("Se esperaba error después de borrar")
	}
}

// TestProcesarPeticion_SuspenderYReasumir prueba el cambio de estado.
func TestProcesarPeticion_SuspenderYReasumir(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	// Suspender
	suspReq := Peticion{
		Servicio:  "gesprog",
		Operacion: "Suspender",
	}
	suspJSON, _ := json.Marshal(suspReq)
	respSusp := ProcesarPeticion(suspJSON)
	var suspResp Respuesta
	json.Unmarshal(respSusp, &suspResp)
	if suspResp.Estado != "ok" {
		t.Fatalf("Suspender falló: %s", suspResp.Mensaje)
	}

	// Leer debe seguir funcionando
	leerReq := Peticion{
		Servicio:  "gesprog",
		Operacion: "Leer",
	}
	leerJSON, _ := json.Marshal(leerReq)
	respLeer := ProcesarPeticion(leerJSON)
	var leerResp Respuesta
	json.Unmarshal(respLeer, &leerResp)
	if leerResp.Estado != "ok" {
		t.Fatal("Leer no funciona en suspendido")
	}

	// Guardar debe fallar
	guardarReq := Peticion{
		Servicio:   "gesprog",
		Operacion:  "Guardar",
		Ejecutable: "/tmp/falso",
	}
	guardarJSON, _ := json.Marshal(guardarReq)
	respGuardar := ProcesarPeticion(guardarJSON)
	var guardarResp Respuesta
	json.Unmarshal(respGuardar, &guardarResp)
	if guardarResp.Estado != "error" || guardarResp.Mensaje != "servicio suspendido" {
		t.Errorf("Se esperaba 'servicio suspendido', got %+v", guardarResp)
	}

	// Reasumir
	reasReq := Peticion{
		Servicio:  "gesprog",
		Operacion: "Reasumir",
	}
	reasJSON, _ := json.Marshal(reasReq)
	respReas := ProcesarPeticion(reasJSON)
	var reasResp Respuesta
	json.Unmarshal(respReas, &reasResp)
	if reasResp.Estado != "ok" {
		t.Fatalf("Reasumir falló: %s", reasResp.Mensaje)
	}

	// Ahora Guardar debe fallar por ejecutable falso, no por estado
	respGuardar2 := ProcesarPeticion(guardarJSON)
	json.Unmarshal(respGuardar2, &guardarResp)
	if guardarResp.Estado == "ok" || guardarResp.Mensaje == "servicio suspendido" {
		t.Error("Después de reasumir, el error no debería ser por suspensión")
	}
}

// TestProcesarPeticion_Terminar prueba la finalización del servicio.
func TestProcesarPeticion_Terminar(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	termReq := Peticion{
		Servicio:  "gesprog",
		Operacion: "Terminar",
	}
	termJSON, _ := json.Marshal(termReq)
	respTerm := ProcesarPeticion(termJSON)
	var termResp Respuesta
	json.Unmarshal(respTerm, &termResp)
	if termResp.Estado != "ok" {
		t.Fatalf("Terminar falló: %s", termResp.Mensaje)
	}

	if !EstaTerminado() {
		t.Error("El estado no se marcó como Terminado")
	}

	// Cualquier otra operación debe fallar
	leerReq := Peticion{
		Servicio:  "gesprog",
		Operacion: "Leer",
	}
	leerJSON, _ := json.Marshal(leerReq)
	respLeer := ProcesarPeticion(leerJSON)
	var leerResp Respuesta
	json.Unmarshal(respLeer, &leerResp)
	if leerResp.Estado != "error" || leerResp.Mensaje != "servicio suspendido" {
		t.Errorf("Después de Terminar debería responder 'servicio suspendido', got %+v", leerResp)
	}
}

// TestProcesarPeticion_OperacionDesconocida prueba el manejo de operación inválida.
func TestProcesarPeticion_OperacionDesconocida(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	req := Peticion{
		Servicio:  "gesprog",
		Operacion: "Volar",
	}
	reqJSON, _ := json.Marshal(req)
	respBytes := ProcesarPeticion(reqJSON)
	var resp Respuesta
	json.Unmarshal(respBytes, &resp)

	if resp.Estado != "error" || resp.Mensaje != "operacion desconocida" {
		t.Errorf("Respuesta incorrecta: %+v", resp)
	}
}

// TestProcesarPeticion_MalJSON prueba el error con JSON inválido.
func TestProcesarPeticion_MalJSON(t *testing.T) {
	cleanup := setupOperaciones(t)
	defer cleanup()

	malJSON := []byte(`{servicio:gesprog,operacion:Guardar}`) // falta comillas
	respBytes := ProcesarPeticion(malJSON)
	var resp Respuesta
	json.Unmarshal(respBytes, &resp)

	if resp.Estado != "error" || resp.Mensaje != "operacion desconocida" {
		// Nota: el código actual devuelve "operacion desconocida" para cualquier error de parseo.
		t.Errorf("Se esperaba error de parseo, got %+v", resp)
	}
}
