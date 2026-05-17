// Package main implementa el servicio ejecutor de procesos de lotes.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// rutaAralmac se inicializa desde main.go con el flag -x.
var rutaAralmac string

// InicializarAlmacenamiento configura la ruta base del aralmac y crea
// los directorios necesarios para el ejecutor.
func InicializarAlmacenamiento(ruta string) error {
	rutaAralmac = ruta
	dirEjecuciones := filepath.Join(rutaAralmac, "ejecuciones")
	if err := os.MkdirAll(dirEjecuciones, 0755); err != nil {
		return fmt.Errorf("InicializarAlmacenamiento: %w", err)
	}
	return nil
}

// metadataPrograma representa los metadatos guardados por gesprog.
// Debe coincidir con la estructura Programa de gesprog/almacenamiento.go.
type metadataPrograma struct {
	ID     string   `json:"id-programa"`
	Nombre string   `json:"nombre"`
	Args   []string `json:"args"`
	Env    []string `json:"env"`
}

// VerificarPrograma comprueba que el id-programa existe en aralmac
// y retorna la ruta absoluta del binario y sus metadatos.
func VerificarPrograma(idPrograma string) (rutaBin string, meta metadataPrograma, err error) {
	metaPath := filepath.Join(rutaAralmac, "programas", idPrograma+".json")
	binPath := filepath.Join(rutaAralmac, "programas", idPrograma+".bin")

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return "", metadataPrograma{}, fmt.Errorf("programa no encontrado")
	}
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return "", metadataPrograma{}, fmt.Errorf("programa no encontrado")
	}

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return "", metadataPrograma{}, fmt.Errorf("error leyendo metadatos del programa: %w", err)
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", metadataPrograma{}, fmt.Errorf("error parseando metadatos del programa: %w", err)
	}
	return binPath, meta, nil
}

// VerificarFichero comprueba que el id-fichero existe en aralmac.
// Retorna la ruta absoluta del fichero.
func VerificarFichero(idFichero string) (string, error) {
	ruta := filepath.Join(rutaAralmac, "ficheros", idFichero+".dat")
	if _, err := os.Stat(ruta); os.IsNotExist(err) {
		return "", fmt.Errorf("fichero no encontrado: %s", idFichero)
	}
	return ruta, nil
}

// registroEjecucion es la estructura que se persiste en disco para cada ejecución.
type registroEjecucion struct {
	IDEjecucion  string `json:"id-ejecucion"`
	IDPrograma   string `json:"id-programa"`
	EstadoStr    string `json:"proceso-estado"`
	CodigoSalida int    `json:"codigo-salida,omitempty"`
	Terminado    bool   `json:"terminado"`
}

// GuardarEjecucion persiste el estado de una ejecución en aralmac/ejecuciones/.
func GuardarEjecucion(info *InfoProceso) error {
	info.mu.RLock()
	reg := registroEjecucion{
		IDEjecucion:  info.IDEjecucion,
		IDPrograma:   info.IDPrograma,
		EstadoStr:    string(info.Estado),
		CodigoSalida: info.CodigoSalida,
		Terminado:    info.Terminado,
	}
	info.mu.RUnlock()

	ruta := filepath.Join(rutaAralmac, "ejecuciones", info.IDEjecucion+".json")
	data, err := json.MarshalIndent(reg, "", "  ")
	if err != nil {
		return fmt.Errorf("GuardarEjecucion: %w", err)
	}
	return os.WriteFile(ruta, data, 0644)
}

// generarIDEjecucion produce un nuevo ID e-XXXX.
// TODO: reemplazar por common.GenerarIDEjecucion() cuando common/ids.go esté listo.
func generarIDEjecucion() (string, error) {
	return "e-0001", nil
}
