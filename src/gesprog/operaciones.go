package main

import "encoding/json"

// Peticion representa el mensaje JSON entrante desde ctrllt.
type Peticion struct {
	Servicio   string   `json:"servicio"`
	Operacion  string   `json:"operacion"`
	IDPrograma string   `json:"id-programa,omitempty"`
	Ejecutable string   `json:"ejecutable,omitempty"`
	Args       []string `json:"args,omitempty"`
	Env        []string `json:"env,omitempty"`
	Ruta       string   `json:"ruta,omitempty"`
}

// Respuesta representa el mensaje JSON de salida hacia ctrllt.
type Respuesta struct {
	Estado     string    `json:"estado"`
	IDPrograma string    `json:"id-programa,omitempty"`
	Programa   *Programa `json:"programa,omitempty"`
	Programas  []string  `json:"programas,omitempty"`
	Mensaje    string    `json:"mensaje,omitempty"`
}

// ProcesarPeticion recibe el JSON crudo, ejecuta la operación y retorna el JSON de respuesta.
func ProcesarPeticion(lineaJSON []byte) []byte {
	var pet Peticion
	if err := json.Unmarshal(lineaJSON, &pet); err != nil {
		return errorJSON("operacion desconocida")
	}

	switch pet.Operacion {
	case "Guardar":
		return opGuardar(pet)
	case "Leer":
		return opLeer(pet)
	case "Actualizar":
		return opActualizar(pet)
	case "Borrar":
		return opBorrar(pet)
	case "Suspender":
		return opSuspender()
	case "Reasumir":
		return opReasumir()
	case "Terminar":
		return opTerminar()
	default:
		return errorJSON("operacion desconocida")
	}
}

// --- operaciones individuales ---

// opGuardar registra un nuevo programa ejecutable.
func opGuardar(pet Peticion) []byte {
	if pet.Ejecutable == "" {
		return errorJSON("falta campo: ejecutable")
	}
	if !EstaActivo() {
		return errorJSON("servicio suspendido")
	}
	id, err := Guardar(pet.Ejecutable, pet.Args, pet.Env)
	if err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{
		Estado:     "ok",
		IDPrograma: id,
	})
}

// opLeer devuelve metadatos de un programa o lista todos los IDs.
func opLeer(pet Peticion) []byte {
	if !EstaDisponibleParaLeer() {
		return errorJSON("servicio suspendido")
	}
	// Leer todos
	if pet.IDPrograma == "" {
		ids, err := ListarTodos()
		if err != nil {
			return errorJSON(err.Error())
		}
		return okJSON(Respuesta{
			Estado:    "ok",
			Programas: ids,
		})
	}
	// Leer por ID
	prog, err := LeerPorID(pet.IDPrograma)
	if err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{
		Estado:   "ok",
		Programa: prog,
	})
}

// opActualizar reemplaza el ejecutable de un programa existente.
func opActualizar(pet Peticion) []byte {
	if !EstaActivo() {
		return errorJSON("servicio suspendido")
	}
	if pet.IDPrograma == "" || pet.Ruta == "" {
		return errorJSON("faltan campos: id-programa, ruta")
	}
	if err := ActualizarRuta(pet.IDPrograma, pet.Ruta); err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{Estado: "ok"})
}

// opBorrar elimina un programa del aralmac.
func opBorrar(pet Peticion) []byte {
	if !EstaActivo() {
		return errorJSON("servicio suspendido")
	}
	if pet.IDPrograma == "" {
		return errorJSON("falta campo: id-programa")
	}
	if err := Borrar(pet.IDPrograma); err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{Estado: "ok"})
}

// opSuspender pausa el servicio.
func opSuspender() []byte {
	if err := Suspender(); err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{Estado: "ok"})
}

// opReasumir reanuda el servicio tras una suspensión.
func opReasumir() []byte {
	if err := Reanudar(); err != nil {
		return errorJSON(err.Error())
	}
	return okJSON(Respuesta{Estado: "ok"})
}

// opTerminar finaliza el servicio de forma ordenada.
func opTerminar() []byte {
	if err := Terminar(); err != nil {
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
