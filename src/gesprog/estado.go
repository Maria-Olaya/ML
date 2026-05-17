package main

import (
	"fmt"
	"sync"
)

// EstadoServicio representa los posibles estados del servicio gesprog.
type EstadoServicio int

const (
	estadoInicio     EstadoServicio = iota
	estadoCorriendo                 // acepta todas las operaciones
	estadoSuspendido                // solo permite Leer
	estadoTerminado
)

// maquinaEstados controla el estado actual del servicio.
type maquinaEstados struct {
	mu      sync.RWMutex
	current EstadoServicio
}

var estado = &maquinaEstados{current: estadoCorriendo}

// EstaActivo informa si el servicio puede atender operaciones de escritura.
func EstaActivo() bool {
	estado.mu.RLock()
	defer estado.mu.RUnlock()
	return estado.current == estadoCorriendo
}

// EstaDisponibleParaLeer informa si el servicio permite la operación Leer.
// En gesprog, Leer está permitida también en estado Suspendido.
func EstaDisponibleParaLeer() bool {
	estado.mu.RLock()
	defer estado.mu.RUnlock()
	return estado.current == estadoCorriendo || estado.current == estadoSuspendido
}

// Suspender intenta pasar el servicio a estado Suspendido.
func Suspender() error {
	estado.mu.Lock()
	defer estado.mu.Unlock()
	if estado.current != estadoCorriendo {
		return fmt.Errorf("transicion invalida")
	}
	estado.current = estadoSuspendido
	return nil
}

// Reanudar intenta pasar el servicio de Suspendido a Corriendo.
func Reanudar() error {
	estado.mu.Lock()
	defer estado.mu.Unlock()
	if estado.current != estadoSuspendido {
		return fmt.Errorf("transicion invalida")
	}
	estado.current = estadoCorriendo
	return nil
}

// Terminar intenta pasar el servicio a estado Terminado.
func Terminar() error {
	estado.mu.Lock()
	defer estado.mu.Unlock()
	if estado.current == estadoTerminado {
		return fmt.Errorf("transicion invalida")
	}
	estado.current = estadoTerminado
	return nil
}

// EstaTerminado informa si el servicio ya finalizó.
func EstaTerminado() bool {
	estado.mu.RLock()
	defer estado.mu.RUnlock()
	return estado.current == estadoTerminado
}
