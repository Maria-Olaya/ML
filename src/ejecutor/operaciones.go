// Package main implementa el servicio ejecutor de procesos de lotes.
package main

import "encoding/json"

// Peticion representa el mensaje JSON entrante desde ctrllt.
type Peticion struct {
	Servicio    string `json:"servicio"`
	Operacion   string `json:"operacion"`
	IDPrograma  string `json:"id-programa,omitempty"`
	IDEjecucion string `json:"id-ejecucion,omitempty"`
	Stdin       string `json:"stdin,omitempty"`
	Stdout      string `json:"stdout,omitempty"`
	Stderr      string `json:"stderr,omitempty"`
}

// RespuestaProceso representa el estado de un proceso en la respuesta JSON.
type RespuestaProceso struct {
	IDEjecucion   string `json:"id-ejecucion"`
	IDPrograma    string `json:"id-programa"`
	ProcesoEstado string `json:"proceso-estado"`
	CodigoSalida  int    `json:"codigo-salida,omitempty"`
}

// Respuesta representa el mensaje JSON de salida hacia ctrllt.
type Respuesta struct {
	Estado      string             `json:"estado"`
	IDEjecucion string             `json:"id-ejecucion,omitempty"`
	Proceso     *RespuestaProceso  `json:"proceso,omitempty"`
	Procesos    []RespuestaProceso `json:"procesos,omitempty"`
	Mensaje     string             `json:"mensaje,omitempty"`
}

// ProcesarPeticion recibe el JSON crudo, ejecuta la operación y retorna el JSON de respuesta.
func ProcesarPeticion(lineaJSON []byte) []byte {
	var pet Peticion
	if err := json.Unmarshal(lineaJSON, &pet); err != nil {
		return errorJSON("operacion desconocida")
	}

	// Verificar que el servicio no está terminado
	if ServicioEstaTerminado() {
		return errorJSON("servicio parando")
	}

	switch pet.Operacion {
	case "Ejecutar":
		return opEjecutar(pet)
	case "Estado":
		return opEstado(pet)
	case "Matar":
		return opMatar(pet)
	case "Suspender":
		return opSuspender()
	case "Reasumir":
		return opReasumir()
	case "Parar":
		return opParar()
	default:
		return errorJSON("operacion desconocida")
	}
}

// --- operaciones individuales ---

// opEjecutar lanza un nuevo proceso de lotes en background.
func opEjecutar(pet Peticion) []byte {
	if !ServicioAceptaEjecuciones() {
		if ServicioEstaTerminado() {
			return errorJSON("servicio parando")
		}
		return errorJSON("servicio suspendido")
	}
	if pet.IDPrograma == "" {
		return errorJSON("falta campo: id-programa")
	}
	idEjecucion, err := LanzarProceso(pet.IDPrograma, pet.Stdin, pet.Stdout, pet.Stderr)
	if err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{
		Estado:      "ok",
		IDEjecucion: idEjecucion,
	})
}

// opEstado consulta el estado de un proceso o lista todos.
func opEstado(pet Peticion) []byte {
	if !ServicioAceptaPeticiones() {
		return errorJSON("servicio parando")
	}

	// Estado de todos los procesos
	if pet.IDEjecucion == "" {
		lista := ListarProcesos()
		respuestas := make([]RespuestaProceso, 0, len(lista))
		for _, info := range lista {
			info.mu.RLock()
			r := RespuestaProceso{
				IDEjecucion:   info.IDEjecucion,
				IDPrograma:    info.IDPrograma,
				ProcesoEstado: string(info.Estado),
			}
			if info.Terminado {
				r.CodigoSalida = info.CodigoSalida
			}
			info.mu.RUnlock()
			respuestas = append(respuestas, r)
		}
		return okJSON(Respuesta{
			Estado:   "ok",
			Procesos: respuestas,
		})
	}

	// Estado de un proceso específico
	info, err := ObtenerProceso(pet.IDEjecucion)
	if err != nil {
		return errorJSON(err.Error())
	}
	info.mu.RLock()
	r := &RespuestaProceso{
		IDEjecucion:   info.IDEjecucion,
		IDPrograma:    info.IDPrograma,
		ProcesoEstado: string(info.Estado),
	}
	if info.Terminado {
		r.CodigoSalida = info.CodigoSalida
	}
	info.mu.RUnlock()
	return okJSON(Respuesta{
		Estado:  "ok",
		Proceso: r,
	})
}

// opMatar termina forzosamente un proceso de lotes.
func opMatar(pet Peticion) []byte {
	if !ServicioAceptaPeticiones() {
		return errorJSON("servicio parando")
	}
	if pet.IDEjecucion == "" {
		return errorJSON("falta campo: id-ejecucion")
	}
	if err := MatarProceso(pet.IDEjecucion); err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{Estado: "ok"})
}

// opSuspender pausa el servicio (los procesos activos siguen corriendo).
func opSuspender() []byte {
	if err := SuspenderServicio(); err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{Estado: "ok"})
}

// opReasumir reactiva el servicio desde estado Suspendidos.
func opReasumir() []byte {
	if err := ReanudarServicio(); err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{Estado: "ok"})
}

// opParar ordena al servicio dejar de aceptar ejecuciones y terminar
// cuando todos los procesos activos finalicen.
func opParar() []byte {
	if err := PararServicio(); err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{Estado: "ok"})
}

// --- helpers ---

// okJSON serializa una Respuesta exitosa.
func okJSON(r Respuesta) []byte {
	data, _ := json.Marshal(r)
	return data
}

// errorJSON construye una respuesta de error con el mensaje dado.
func errorJSON(mensaje string) []byte {
	data, _ := json.Marshal(Respuesta{
		Estado:  "error",
		Mensaje: mensaje,
	})
	return data
}
