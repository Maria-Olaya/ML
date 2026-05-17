// Package main implementa el servicio ejecutor de procesos de lotes.
package main

import (
	"fmt"
	"os/exec"
	"sync"
)

// --- Estado del servicio ejecutor ---

// EstadoServicio representa los posibles estados del servicio ejecutor.
type EstadoServicio int

const (
	estadoEjecutar    EstadoServicio = iota // acepta nuevas ejecuciones
	estadoSuspendidos                       // procesos siguen corriendo, rechaza nuevos Ejecutar
	estadoParando                           // espera que terminen los procesos activos
	estadoTerminado                         // servicio finalizado
)

// maquinaServicio controla el estado actual del servicio.
type maquinaServicio struct {
	mu      sync.RWMutex
	current EstadoServicio
}

var servicio = &maquinaServicio{current: estadoEjecutar}

// ServicioAceptaEjecuciones informa si el servicio acepta nuevas peticiones Ejecutar.
// Solo es true en estado Ejecutar.
func ServicioAceptaEjecuciones() bool {
	servicio.mu.RLock()
	defer servicio.mu.RUnlock()
	return servicio.current == estadoEjecutar
}

// ServicioAceptaPeticiones informa si el servicio acepta peticiones que no sean Ejecutar.
// Estado, Matar, Suspender, Reasumir y Parar se aceptan en Ejecutar y Suspendidos.
// ServicioAceptaPeticiones informa si el servicio acepta peticiones que no sean Ejecutar.
// Estado, Matar y Parar se aceptan en Ejecutar, Suspendidos y Parando.
func ServicioAceptaPeticiones() bool {
	servicio.mu.RLock()
	defer servicio.mu.RUnlock()
	return servicio.current == estadoEjecutar ||
		servicio.current == estadoSuspendidos ||
		servicio.current == estadoParando
}

// ServicioEstaTerminado informa si el servicio ya finalizó.
func ServicioEstaTerminado() bool {
	servicio.mu.RLock()
	defer servicio.mu.RUnlock()
	return servicio.current == estadoTerminado
}

// SuspenderServicio pasa el servicio a Suspendidos.
// Los procesos activos siguen corriendo pero no se aceptan nuevos Ejecutar.
func SuspenderServicio() error {
	servicio.mu.Lock()
	defer servicio.mu.Unlock()
	if servicio.current != estadoEjecutar {
		return fmt.Errorf("transicion invalida")
	}
	servicio.current = estadoSuspendidos
	return nil
}

// ReanudarServicio vuelve de Suspendidos a Ejecutar.
func ReanudarServicio() error {
	servicio.mu.Lock()
	defer servicio.mu.Unlock()
	if servicio.current != estadoSuspendidos {
		return fmt.Errorf("transicion invalida")
	}
	servicio.current = estadoEjecutar
	return nil
}

// PararServicio pasa el servicio a Parando.
// Solo válido desde Ejecutar (no desde Suspendidos, según el diagrama).
func PararServicio() error {
	servicio.mu.Lock()
	defer servicio.mu.Unlock()
	if servicio.current != estadoEjecutar {
		return fmt.Errorf("transicion invalida")
	}
	servicio.current = estadoParando
	return nil
}

// TerminarServicio finaliza el servicio. Se llama automáticamente
// cuando el contador de procesos activos llega a cero estando en Parando.
func TerminarServicio() {
	servicio.mu.Lock()
	defer servicio.mu.Unlock()
	servicio.current = estadoTerminado
}

// --- Estado de cada proceso individual ---

// EstadoProceso representa el estado de un proceso de lotes.
type EstadoProceso string

const (
	ProcesoEjecutando EstadoProceso = "Ejecutando"
	ProcesoTerminado  EstadoProceso = "Terminado"
)

// InfoProceso contiene el estado y resultado de un proceso de lotes.
// El mutex protege el acceso concurrente entre la goroutine monitor
// y las peticiones de Estado/Matar.
type InfoProceso struct {
	mu           sync.RWMutex
	IDEjecucion  string
	IDPrograma   string
	Estado       EstadoProceso
	CodigoSalida int
	Terminado    bool
	Cmd          *exec.Cmd // referencia al proceso del SO, para poder matarlo
}

// registroProcesos almacena todos los procesos lanzados, indexados por id-ejecucion.
// El mutex protege el mapa contra accesos concurrentes.
var (
	muProcesos       sync.RWMutex
	registroProcesos = make(map[string]*InfoProceso)
	contadorActivos  int // procesos en estado Ejecutando
)

// RegistrarProceso agrega un nuevo proceso al registro.
func RegistrarProceso(idEjecucion, idPrograma string) *InfoProceso {
	info := &InfoProceso{
		IDEjecucion: idEjecucion,
		IDPrograma:  idPrograma,
		Estado:      ProcesoEjecutando,
	}
	muProcesos.Lock()
	registroProcesos[idEjecucion] = info
	contadorActivos++
	muProcesos.Unlock()
	return info
}

// MarcarTerminado actualiza el estado de un proceso cuando su goroutine monitor detecta que terminó.
// Si el servicio está en Parando y no quedan procesos activos, auto-termina el servicio.
func MarcarTerminado(idEjecucion string, codigoSalida int) {
	muProcesos.Lock()
	info, existe := registroProcesos[idEjecucion]
	if existe && !info.Terminado {
		info.mu.Lock()
		info.Estado = ProcesoTerminado
		info.CodigoSalida = codigoSalida
		info.Terminado = true
		info.mu.Unlock()
		contadorActivos--
	}
	activos := contadorActivos
	muProcesos.Unlock()

	// Si estamos en Parando y no quedan procesos activos, terminar el servicio.
	if activos == 0 {
		servicio.mu.RLock()
		parando := servicio.current == estadoParando
		servicio.mu.RUnlock()
		if parando {
			TerminarServicio()
		}
	}
}

// ObtenerProceso retorna la info de un proceso por su id-ejecucion.
func ObtenerProceso(idEjecucion string) (*InfoProceso, error) {
	muProcesos.RLock()
	defer muProcesos.RUnlock()
	info, existe := registroProcesos[idEjecucion]
	if !existe {
		return nil, fmt.Errorf("proceso no encontrado")
	}
	return info, nil
}

// ListarProcesos retorna una copia del estado de todos los procesos registrados.
func ListarProcesos() []*InfoProceso {
	muProcesos.RLock()
	defer muProcesos.RUnlock()
	lista := make([]*InfoProceso, 0, len(registroProcesos))
	for _, info := range registroProcesos {
		lista = append(lista, info)
	}
	return lista
}
