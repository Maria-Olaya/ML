package main

import (
	"fmt"
	"syscall"
)

func main() {
	pipeName := `\\.\pipe\gesprog_pipe`

	// Abrir el pipe con CreateFile
	pathPtr, err := syscall.UTF16PtrFromString(pipeName)
	if err != nil {
		fmt.Println("Error convirtiendo nombre:", err)
		return
	}

	handle, err := syscall.CreateFile(
		pathPtr,
		syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		0,
		nil,
		syscall.OPEN_EXISTING,
		0,
		0,
	)
	if err != nil {
		fmt.Println("Error conectando al pipe:", err)
		return
	}
	defer syscall.CloseHandle(handle)

	// Enviar petición Guardar
	msg := `{"servicio":"gesprog","operacion":"Guardar","ejecutable":"C:\\Windows\\System32\\calc.exe"}` + "\n"
	data := []byte(msg)
	var written uint32
	err = syscall.WriteFile(handle, data, &written, nil)
	if err != nil {
		fmt.Println("Error escribiendo:", err)
		return
	}

	// Leer respuesta
	buf := make([]byte, 4096)
	var read uint32
	err = syscall.ReadFile(handle, buf, &read, nil)
	if err != nil {
		fmt.Println("Error leyendo:", err)
		return
	}
	fmt.Println("Respuesta:", string(buf[:read]))
}
