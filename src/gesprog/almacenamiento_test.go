package main

import (
	"path/filepath"
	"testing"
)

// setupCrea un directorio temporal e inicializa el almacenamiento.
// Devuelve la ruta base y una función cleanup.
func setup(t *testing.T) (string, func()) {
	tempDir := t.TempDir() // se borra automáticamente al final
	err := InicializarAlmacenamiento(tempDir)
	if err != nil {
		t.Fatalf("InicializarAlmacenamiento falló: %v", err)
	}
	return tempDir, func() {
		// cleanup adicional si se necesita, pero TempDir lo hace solo.
	}
}

// crearEjecutableTemporal crea un archivo ejecutable simulado en un directorio temporal.
/*func crearEjecutableTemporal(t *testing.T, nombre string) string {
	dir := t.TempDir()
	ruta := filepath.Join(dir, nombre)
	contenido := []byte("#!/bin/bash\necho 'hola mundo'")
	err := os.WriteFile(ruta, contenido, 0755)
	if err != nil {
		t.Fatalf("no se pudo crear ejecutable temporal: %v", err)
	}
	return ruta
}*/

func TestGuardarYLeer(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	ejecutable := crearEjecutableTemporal(t, "mi_programa")
	args := []string{"-l", "-a"}
	env := []string{"PATH=/usr/bin", "LANG=es"}

	id, err := Guardar(ejecutable, args, env)
	if err != nil {
		t.Fatalf("Guardar falló: %v", err)
	}
	if id != "p-0001" { // mientras sea placeholder
		t.Errorf("ID esperado p-0001, got %s", id)
	}

	prog, err := LeerPorID(id)
	if err != nil {
		t.Fatalf("LeerPorID falló: %v", err)
	}
	if prog.Nombre != "mi_programa" {
		t.Errorf("Nombre incorrecto: esperado 'mi_programa', got %s", prog.Nombre)
	}
	if len(prog.Args) != 2 || prog.Args[0] != "-l" {
		t.Errorf("Args incorrectos: %v", prog.Args)
	}
	if len(prog.Env) != 2 || prog.Env[1] != "LANG=es" {
		t.Errorf("Env incorrectos: %v", prog.Env)
	}
}

func TestListarTodos(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	// Guardar un programa
	ejecutable := crearEjecutableTemporal(t, "programa1")
	Guardar(ejecutable, nil, nil)

	ids, err := ListarTodos()
	if err != nil {
		t.Fatalf("ListarTodos falló: %v", err)
	}
	if len(ids) != 1 || ids[0] != "p-0001" {
		t.Errorf("ListarTodos esperado [p-0001], got %v", ids)
	}
}

func TestActualizarRuta(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	ejecutable1 := crearEjecutableTemporal(t, "v1")
	id, err := Guardar(ejecutable1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	// Nuevo ejecutable
	ejecutable2 := crearEjecutableTemporal(t, "v2")
	err = ActualizarRuta(id, ejecutable2)
	if err != nil {
		t.Fatalf("ActualizarRuta falló: %v", err)
	}

	prog, _ := LeerPorID(id)
	if prog.Nombre != "v2" {
		t.Errorf("Nombre no actualizado: esperado 'v2', got %s", prog.Nombre)
	}
}

func TestBorrar(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	ejecutable := crearEjecutableTemporal(t, "para_borrar")
	id, err := Guardar(ejecutable, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	err = Borrar(id)
	if err != nil {
		t.Fatalf("Borrar falló: %v", err)
	}

	_, err = LeerPorID(id)
	if err == nil {
		t.Error("Se esperaba error después de borrar")
	}
}

func TestRutaBinario(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	ejecutable := crearEjecutableTemporal(t, "binario")
	id, err := Guardar(ejecutable, nil, nil)
	if err != nil {
		t.Fatal(err)
	}

	ruta, err := RutaBinario(id)
	if err != nil {
		t.Fatalf("RutaBinario falló: %v", err)
	}
	esperado := filepath.Join(rutaAralmac, id+".bin")
	if ruta != esperado {
		t.Errorf("Ruta incorrecta: esperado %s, got %s", esperado, ruta)
	}
}

func TestGuardarErrorEjecutableNoExiste(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	_, err := Guardar("/ruta/inexistente", nil, nil)
	if err == nil {
		t.Error("Se esperaba error por ejecutable inexistente")
	}
}

func TestGuardarErrorRutaEsDirectorio(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	dir := t.TempDir()
	_, err := Guardar(dir, nil, nil)
	if err == nil {
		t.Error("Se esperaba error por ser directorio")
	}
}

func TestLeerPorIDNoExistente(t *testing.T) {
	_, cleanup := setup(t)
	defer cleanup()

	_, err := LeerPorID("p-0999")
	if err == nil {
		t.Error("Se esperaba error para ID inexistente")
	}
}
