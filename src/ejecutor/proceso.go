// Package main implementa el servicio ejecutor de procesos de lotes.
package main

import (
	"fmt"
	"os"
	"os/exec"
)

// LanzarProceso inicia un proceso de lotes en background.
// Verifica que el programa y los ficheros existan, lanza el proceso
// y devuelve el id-ejecucion inmediatamente sin esperar a que termine.
func LanzarProceso(idPrograma, idStdin, idStdout, idStderr string) (string, error) {
	// Verificar que el programa existe en aralmac
	rutaBin, meta, err := VerificarPrograma(idPrograma)
	if err != nil {
		return "", fmt.Errorf("no se pudo ejecutar el programa: %w", err)
	}

	// Generar id-ejecucion
	idEjecucion, err := generarIDEjecucion()
	if err != nil {
		return "", fmt.Errorf("no se pudo ejecutar el programa: %w", err)
	}

	// Construir el comando con args y env del programa
	cmd := exec.Command(rutaBin, meta.Args...)
	cmd.Env = append(os.Environ(), meta.Env...)

	// archivosAbiertos guarda los archivos abiertos para cerrarlos
	// en la goroutine monitor, después de que el proceso termine.
	var archivosAbiertos []*os.File

	// Redirigir stdin si se especificó un fichero
	if idStdin != "" {
		rutaStdin, err := VerificarFichero(idStdin)
		if err != nil {
			return "", err
		}
		f, err := os.Open(rutaStdin)
		if err != nil {
			return "", fmt.Errorf("no se pudo abrir stdin: %w", err)
		}
		cmd.Stdin = f
		archivosAbiertos = append(archivosAbiertos, f)
	}

	// Redirigir stdout si se especificó un fichero
	if idStdout != "" {
		rutaStdout, err := VerificarFichero(idStdout)
		if err != nil {
			return "", err
		}
		f, err := os.OpenFile(rutaStdout, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return "", fmt.Errorf("no se pudo abrir stdout: %w", err)
		}
		cmd.Stdout = f
		archivosAbiertos = append(archivosAbiertos, f)
	}

	// Redirigir stderr si se especificó un fichero
	if idStderr != "" {
		rutaStderr, err := VerificarFichero(idStderr)
		if err != nil {
			return "", err
		}
		f, err := os.OpenFile(rutaStderr, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			return "", fmt.Errorf("no se pudo abrir stderr: %w", err)
		}
		cmd.Stderr = f
		archivosAbiertos = append(archivosAbiertos, f)
	}

	// Registrar el proceso antes de lanzarlo
	info := RegistrarProceso(idEjecucion, idPrograma)

	// Guardar referencia al cmd para poder matarlo después
	info.mu.Lock()
	info.Cmd = cmd
	info.mu.Unlock()

	// Lanzar el proceso en background
	if err := cmd.Start(); err != nil {
		for _, f := range archivosAbiertos {
			f.Close()
		}
		MarcarTerminado(idEjecucion, -1)
		return "", fmt.Errorf("no se pudo ejecutar el programa: %w", err)
	}

	// Guardar estado inicial en disco
	_ = GuardarEjecucion(info)

	// Goroutine monitor: espera que el proceso termine,
	// cierra los archivos y actualiza el estado.
	go func() {
		err := cmd.Wait()

		// Cerrar archivos después de que el proceso terminó
		for _, f := range archivosAbiertos {
			f.Close()
		}

		codigoSalida := 0
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				codigoSalida = exitErr.ExitCode()
			} else {
				codigoSalida = -1
			}
		}
		MarcarTerminado(idEjecucion, codigoSalida)
		_ = GuardarEjecucion(info)
	}()

	return idEjecucion, nil
}

// MatarProceso termina forzosamente un proceso de lotes en ejecución.
func MatarProceso(idEjecucion string) error {
	info, err := ObtenerProceso(idEjecucion)
	if err != nil {
		return fmt.Errorf("proceso no encontrado")
	}

	info.mu.RLock()
	terminado := info.Terminado
	cmd := info.Cmd
	info.mu.RUnlock()

	if terminado || cmd == nil || cmd.Process == nil {
		return fmt.Errorf("proceso no encontrado o ya terminado")
	}

	if err := cmd.Process.Kill(); err != nil {
		return fmt.Errorf("no se pudo matar el proceso: %w", err)
	}
	return nil
}
