package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// rutaAralmac se inicializa desde main.go con el flag -x.
var rutaAralmac string

// InicializarAlmacenamiento configura la ruta base del aralmac.
// Debe llamarse desde main.go antes de arrancar el servidor.
func InicializarAlmacenamiento(ruta string) error {
	rutaAralmac = filepath.Join(ruta, "programas")
	if err := os.MkdirAll(rutaAralmac, 0755); err != nil {
		return fmt.Errorf("InicializarAlmacenamiento: %w", err)
	}
	return nil
}

// Programa almacena la información de un ejecutable registrado.
type Programa struct {
	ID     string   `json:"id-programa"`
	Nombre string   `json:"nombre"`
	Args   []string `json:"args"`
	Env    []string `json:"env"`
}

// Guardar copia el binario al aralmac y guarda sus metadatos.
// Retorna el id-programa asignado.
func Guardar(ejecutablePath string, args, env []string) (string, error) {
	// Verificar que el ejecutable existe y no es un directorio
	info, err := os.Stat(ejecutablePath)
	if err != nil {
		return "", fmt.Errorf("ejecutable no encontrado: %w", err)
	}
	if info.IsDir() {
		return "", errors.New("la ruta es un directorio, no un ejecutable")
	}

	// Generar nuevo ID p-XXXX
	// TODO: reemplazar por common.GenerarIDPrograma() cuando common/ids.go esté listo
	id, err := generarIDPrograma()
	if err != nil {
		return "", fmt.Errorf("no se pudo guardar el programa: %w", err)
	}

	// Copiar binario
	binPath := filepath.Join(rutaAralmac, id+".bin")
	if err := copiarArchivo(ejecutablePath, binPath); err != nil {
		return "", fmt.Errorf("no se pudo guardar el programa: %w", err)
	}
	// Permisos de ejecución (no tiene efecto en Windows)
	_ = os.Chmod(binPath, 0755)

	// Guardar metadatos JSON
	meta := Programa{
		ID:     id,
		Nombre: filepath.Base(ejecutablePath),
		Args:   args,
		Env:    env,
	}
	if err := guardarMetadatos(id, meta); err != nil {
		_ = os.Remove(binPath) // revertir binario si falla metadata
		return "", fmt.Errorf("no se pudo guardar el programa: %w", err)
	}

	return id, nil
}

// LeerPorID devuelve los metadatos del programa con el id dado.
func LeerPorID(id string) (*Programa, error) {
	metaPath := filepath.Join(rutaAralmac, id+".json")
	binPath := filepath.Join(rutaAralmac, id+".bin")

	if _, err := os.Stat(metaPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("programa no encontrado")
	}
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("programa no encontrado")
	}

	data, err := os.ReadFile(metaPath)
	if err != nil {
		return nil, fmt.Errorf("error al leer metadatos: %w", err)
	}
	var prog Programa
	if err := json.Unmarshal(data, &prog); err != nil {
		return nil, fmt.Errorf("error al parsear metadatos: %w", err)
	}
	return &prog, nil
}

// ListarTodos devuelve todos los id-programa registrados.
func ListarTodos() ([]string, error) {
	entradas, err := os.ReadDir(rutaAralmac)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("error al listar programas")
	}
	var ids []string
	for _, e := range entradas {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			ids = append(ids, e.Name()[:len(e.Name())-5])
		}
	}
	return ids, nil
}

// ActualizarRuta reemplaza el ejecutable de un programa existente.
func ActualizarRuta(id, nuevaRuta string) error {
	if _, err := LeerPorID(id); err != nil {
		return err
	}
	info, err := os.Stat(nuevaRuta)
	if err != nil {
		return fmt.Errorf("nuevo ejecutable no encontrado: %w", err)
	}
	if info.IsDir() {
		return errors.New("la ruta es un directorio")
	}
	binPath := filepath.Join(rutaAralmac, id+".bin")
	if err := copiarArchivo(nuevaRuta, binPath); err != nil {
		return fmt.Errorf("no se pudo actualizar el programa: %w", err)
	}
	_ = os.Chmod(binPath, 0755)

	// Actualizar nombre en metadatos
	meta, _ := LeerPorID(id)
	meta.Nombre = filepath.Base(nuevaRuta)
	return guardarMetadatos(id, *meta)
}

// Borrar elimina el binario y los metadatos de un programa.
func Borrar(id string) error {
	binPath := filepath.Join(rutaAralmac, id+".bin")
	metaPath := filepath.Join(rutaAralmac, id+".json")

	_, errBin := os.Stat(binPath)
	_, errMeta := os.Stat(metaPath)
	if os.IsNotExist(errBin) && os.IsNotExist(errMeta) {
		return fmt.Errorf("programa no encontrado")
	}
	_ = os.Remove(binPath)
	_ = os.Remove(metaPath)
	return nil
}

// RutaBinario retorna la ruta absoluta del ejecutable almacenado.
// La usa el ejecutor para lanzar el proceso.
func RutaBinario(id string) (string, error) {
	ruta := filepath.Join(rutaAralmac, id+".bin")
	if _, err := os.Stat(ruta); err != nil {
		return "", fmt.Errorf("programa no encontrado")
	}
	return ruta, nil
}

// --- funciones internas ---

// guardarMetadatos escribe el JSON de un Programa en disco.
func guardarMetadatos(id string, prog Programa) error {
	ruta := filepath.Join(rutaAralmac, id+".json")
	data, err := json.MarshalIndent(prog, "", "  ")
	if err != nil {
		return fmt.Errorf("guardarMetadatos: %w", err)
	}
	return os.WriteFile(ruta, data, 0644)
}

// copiarArchivo copia el contenido de origen a destino.
func copiarArchivo(origen, destino string) error {
	src, err := os.Open(origen)
	if err != nil {
		return fmt.Errorf("copiarArchivo: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destino)
	if err != nil {
		return fmt.Errorf("copiarArchivo: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copiarArchivo: %w", err)
	}
	return nil
}

// generarIDPrograma produce un nuevo ID p-XXXX.
// TODO: reemplazar por common.GenerarIDPrograma() cuando common/ids.go esté listo.
func generarIDPrograma() (string, error) {
	return "p-0001", nil
}
