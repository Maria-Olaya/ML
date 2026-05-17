package main

import (
	"testing"
)

// Resetea el estado a Corriendo para pruebas
func resetEstado() {
	estado.mu.Lock()
	defer estado.mu.Unlock()
	estado.current = estadoCorriendo
}

func TestEstadoInicial(t *testing.T) {
	resetEstado()
	if !EstaActivo() {
		t.Error("Se esperaba activo al inicio")
	}
	if !EstaDisponibleParaLeer() {
		t.Error("Se esperaba disponible para leer")
	}
	if EstaTerminado() {
		t.Error("No debería estar terminado")
	}
}

func TestSuspenderYReanudar(t *testing.T) {
	resetEstado()

	// Suspender
	err := Suspender()
	if err != nil {
		t.Fatalf("Suspender falló: %v", err)
	}
	if EstaActivo() {
		t.Error("No debería estar activo después de suspender")
	}
	if !EstaDisponibleParaLeer() {
		t.Error("Debería permitir Leer incluso suspendido")
	}

	// Reanudar
	err = Reanudar()
	if err != nil {
		t.Fatalf("Reanudar falló: %v", err)
	}
	if !EstaActivo() {
		t.Error("Debería estar activo después de reanudar")
	}
}

func TestTransicionesInvalidas(t *testing.T) {
	resetEstado()

	// Primera suspensión ok
	if err := Suspender(); err != nil {
		t.Fatal(err)
	}
	// Segunda suspensión debe fallar (ya suspendido)
	if err := Suspender(); err == nil {
		t.Error("Suspender dos veces debería fallar")
	}

	// Reanudar ok
	if err := Reanudar(); err != nil {
		t.Fatal(err)
	}
	// Reanudar otra vez debe fallar
	if err := Reanudar(); err == nil {
		t.Error("Reanudar dos veces debería fallar")
	}
}

func TestTerminar(t *testing.T) {
	resetEstado()

	err := Terminar()
	if err != nil {
		t.Fatalf("Terminar falló: %v", err)
	}
	if !EstaTerminado() {
		t.Error("Debería estar terminado")
	}

	// Operaciones después de terminado deben fallar
	if err := Suspender(); err == nil {
		t.Error("Suspender después de terminar debería fallar")
	}
	if err := Reanudar(); err == nil {
		t.Error("Reanudar después de terminar debería fallar")
	}
}
