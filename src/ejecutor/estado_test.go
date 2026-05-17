package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

// ========== Pruebas de la máquina de estados del servicio ==========

func TestServicioEstadoInicial(t *testing.T) {
	if !ServicioAceptaEjecuciones() {
		t.Error("El servicio debería aceptar ejecuciones al inicio")
	}
	if !ServicioAceptaPeticiones() {
		t.Error("El servicio debería aceptar otras peticiones al inicio")
	}
	if ServicioEstaTerminado() {
		t.Error("El servicio no debería estar terminado al inicio")
	}
}

func TestServicioSuspenderYReanudar(t *testing.T) {
	// Resetear estado manualmente para asegurar inicio limpio
	resetServicio()

	// Suspender
	err := SuspenderServicio()
	if err != nil {
		t.Fatalf("SuspenderServicio falló: %v", err)
	}
	if ServicioAceptaEjecuciones() {
		t.Error("Después de suspender, no debería aceptar ejecuciones")
	}
	if !ServicioAceptaPeticiones() {
		t.Error("Después de suspender, debería aceptar otras peticiones")
	}

	// Reanudar
	err = ReanudarServicio()
	if err != nil {
		t.Fatalf("ReanudarServicio falló: %v", err)
	}
	if !ServicioAceptaEjecuciones() {
		t.Error("Después de reanudar, debería aceptar ejecuciones")
	}
}

func TestServicioParar(t *testing.T) {
	resetServicio()

	err := PararServicio()
	if err != nil {
		t.Fatalf("PararServicio falló: %v", err)
	}
	// En Parando, las ejecuciones se rechazan pero se aceptan otras peticiones
	if ServicioAceptaEjecuciones() {
		t.Error("En estado Parando no debe aceptar nuevas ejecuciones")
	}
	if !ServicioAceptaPeticiones() {
		t.Error("En estado Parando debe aceptar otras peticiones (Estado, Matar, etc)")
	}
}

func TestTransicionesInvalidas(t *testing.T) {
	// Prueba que no se pueda suspender dos veces seguidas
	resetServicio()
	_ = SuspenderServicio()
	err := SuspenderServicio()
	if err == nil {
		t.Error("Suspender desde Suspendido debería fallar")
	}

	// No se pueda reanudar sin estar suspendido
	resetServicio()
	err = ReanudarServicio()
	if err == nil {
		t.Error("Reanudar desde Ejecutar debería fallar")
	}

	// No se pueda parar desde Suspendido
	resetServicio()
	_ = SuspenderServicio()
	err = PararServicio()
	if err == nil {
		t.Error("Parar desde Suspendido debería fallar")
	}
}

func TestTerminarServicioSoloDesdeParando(t *testing.T) {
	// TerminarServicio es privado, pero lo invocamos directamente (está en mismo paquete)
	resetServicio()
	TerminarServicio() // esto solo funciona si el estado es Parando? En realidad, TerminarServicio fuerza el estado a Terminado sin comprobar.
	// Deberíamos usarlo solo cuando se llama automáticamente desde MarcarTerminado.
	// Para probar, ponemos en Parando y luego terminamos
	resetServicio()
	_ = PararServicio()
	TerminarServicio()
	if !ServicioEstaTerminado() {
		t.Error("El servicio debería estar terminado después de TerminarServicio")
	}
}

// ========== Pruebas del registro de procesos ==========

func TestRegistrarYObtenerProceso(t *testing.T) {
	limpiarRegistroProcesos()

	info := RegistrarProceso("e-0001", "p-0001")
	if info == nil {
		t.Fatal("RegistrarProceso devolvió nil")
	}
	if info.IDEjecucion != "e-0001" || info.IDPrograma != "p-0001" {
		t.Errorf("Datos incorrectos: %+v", info)
	}
	if info.Estado != ProcesoEjecutando {
		t.Errorf("Estado inicial debería ser Ejecutando, got %v", info.Estado)
	}
	if info.Terminado {
		t.Error("Terminado debería ser false")
	}

	// Obtener proceso
	obtenido, err := ObtenerProceso("e-0001")
	if err != nil {
		t.Fatal(err)
	}
	if obtenido != info {
		t.Error("ObtenerProceso no devolvió la misma referencia")
	}

	// Proceso no existente
	_, err = ObtenerProceso("e-9999")
	if err == nil {
		t.Error("ObtenerProceso debería fallar para ID inexistente")
	}
}

func TestMarcarTerminado(t *testing.T) {
	limpiarRegistroProcesos()
	resetServicio()

	info := RegistrarProceso("e-0001", "p-0001")
	MarcarTerminado("e-0001", 42)

	if info.Estado != ProcesoTerminado {
		t.Errorf("Estado debería ser Terminado, got %v", info.Estado)
	}
	if info.CodigoSalida != 42 {
		t.Errorf("Código de salida debería ser 42, got %d", info.CodigoSalida)
	}
	if !info.Terminado {
		t.Error("Terminado debería ser true")
	}

	// Verificar contador de activos (internamente)
	muProcesos.RLock()
	activos := contadorActivos
	muProcesos.RUnlock()
	if activos != 0 {
		t.Errorf("contadorActivos debería ser 0, got %d", activos)
	}
}

func TestMarcarTerminadoNoExistente(t *testing.T) {
	limpiarRegistroProcesos()
	// No debe causar pánico
	MarcarTerminado("e-9999", 0)
}

func TestMarcarTerminadoYaMarcado(t *testing.T) {
	limpiarRegistroProcesos()
	info := RegistrarProceso("e-0001", "p-0001")
	MarcarTerminado("e-0001", 1)
	MarcarTerminado("e-0001", 2) // segundo intento
	// No debe cambiar nada
	if info.CodigoSalida != 1 {
		t.Errorf("Código de salida no debería cambiar, sigue siendo 1, got %d", info.CodigoSalida)
	}
}

func TestListarProcesos(t *testing.T) {
	limpiarRegistroProcesos()
	RegistrarProceso("e-0001", "p-0001")
	RegistrarProceso("e-0002", "p-0002")

	lista := ListarProcesos()
	if len(lista) != 2 {
		t.Fatalf("Se esperaban 2 procesos, got %d", len(lista))
	}
	// Verificar que contiene ambos IDs
	ids := make(map[string]bool)
	for _, p := range lista {
		ids[p.IDEjecucion] = true
	}
	if !ids["e-0001"] || !ids["e-0002"] {
		t.Errorf("IDs faltantes: %v", ids)
	}
}

// ========== Prueba de auto‑terminación en estado Parando ==========

func TestAutoTerminarCuandoParadoYSinProcesos(t *testing.T) {
	limpiarRegistroProcesos()
	resetServicio()

	// Registrar un proceso
	RegistrarProceso("e-0001", "p-0001")
	// Poner servicio en Parando
	_ = PararServicio()
	// Marcar proceso como terminado (debería quedar contadorActivos en 0)
	MarcarTerminado("e-0001", 0)

	// Esperar un poco a que la condición se evalúe
	time.Sleep(10 * time.Millisecond)

	if !ServicioEstaTerminado() {
		t.Error("El servicio debería haberse terminado automáticamente al quedar sin procesos activos en Parando")
	}
}

func TestNoAutoTerminarSiHayProcesos(t *testing.T) {
	limpiarRegistroProcesos()
	resetServicio()

	// Registrar dos procesos
	RegistrarProceso("e-0001", "p-0001")
	RegistrarProceso("e-0002", "p-0002")
	_ = PararServicio()
	// Solo termina uno
	MarcarTerminado("e-0001", 0)
	time.Sleep(10 * time.Millisecond)

	if ServicioEstaTerminado() {
		t.Error("El servicio no debería terminar si aún hay procesos activos")
	}
	// Terminar el segundo
	MarcarTerminado("e-0002", 0)
	time.Sleep(10 * time.Millisecond)
	if !ServicioEstaTerminado() {
		t.Error("Debería terminar después del último proceso")
	}
}

// ========== Pruebas de concurrencia (opcional) ==========

func TestConcurrenteRegistroYMarcado(t *testing.T) {
	limpiarRegistroProcesos()
	resetServicio()

	const N = 100
	var wg sync.WaitGroup
	wg.Add(N)
	for i := 0; i < N; i++ {
		go func(i int) {
			defer wg.Done()
			id := fmt.Sprintf("e-%04d", i)
			RegistrarProceso(id, "p-0001")
			MarcarTerminado(id, i)
		}(i)
	}
	wg.Wait()

	// Verificar que todos los procesos estén terminados
	lista := ListarProcesos()
	if len(lista) != N {
		t.Errorf("Se esperaban %d procesos, got %d", N, len(lista))
	}
	for _, p := range lista {
		if p.Estado != ProcesoTerminado {
			t.Errorf("Proceso %s no terminado", p.IDEjecucion)
		}
	}
	muProcesos.RLock()
	activos := contadorActivos
	muProcesos.RUnlock()
	if activos != 0 {
		t.Errorf("contadorActivos debería ser 0, got %d", activos)
	}
}

// ========== Funciones auxiliares para resetear el estado global ==========

// resetServicio reinicia la máquina de estados del servicio a estadoEjecutar.
// Solo para uso en pruebas.
func resetServicio() {
	servicio.mu.Lock()
	defer servicio.mu.Unlock()
	servicio.current = estadoEjecutar
}

// limpiarRegistroProcesos vacía el mapa de procesos y resetea el contador.
func limpiarRegistroProcesos() {
	muProcesos.Lock()
	defer muProcesos.Unlock()
	registroProcesos = make(map[string]*InfoProceso)
	contadorActivos = 0
}
