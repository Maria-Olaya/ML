# common — Utilidades compartidas

Este paquete concentra todo el código que reutilizan los demás componentes (`ctrllt`, `gesfich`, `gesprog`, `ejecutor`). No contiene lógica de negocio.

## Archivos previstos

| Archivo      | Responsabilidad                                                                 |
|--------------|---------------------------------------------------------------------------------|
| `json.go`    | Serialización/deserialización de mensajes del protocolo (encode/decode JSON+\n) |
| `pipe.go`    | Creación, apertura, lectura y escritura de named pipes en Linux y Windows 11    |
| `lock.go`    | Bloqueo exclusivo de archivos (`syscall.Flock` en Linux, `LockFile` en Windows) |
| `ids.go`     | Generación atómica de identificadores: `f-XXXX` (ficheros), `p-XXXX` (programas), `l-XXXX` (lotes), `cli-XXXX` (clientes, autogenerado por el cliente) |
| `errors.go`  | Códigos de error estándar del sistema y función de construcción de respuestas de error |

## Notas de portabilidad

- `pipe.go` y `lock.go` usan build tags (`//go:build linux` / `//go:build windows`) para separar las implementaciones por SO sin duplicar la interfaz pública.
- `ids.go` protege el acceso al archivo de secuencia con bloqueo exclusivo a nivel de archivo para garantizar unicidad incluso bajo peticiones concurrentes y tras reinicios del servicio.

## Secuencias de IDs

Cada secuencia se almacena en `aralmac/secuencias/` en los archivos `next_fichero.txt`, `next_programa.txt` y `next_lote.txt`. El procedimiento para obtener un nuevo ID:
1. Abrir el archivo con bloqueo exclusivo (`syscall.Flock` en Linux / `LockFile` en Windows).
2. Leer el número, incrementarlo, escribirlo de vuelta.
3. Liberar el bloqueo.
4. Formatear como `tipo-XXXX` con 4 dígitos (ej. `f-0042`).
5. Si el archivo no existe, crearlo con valor inicial `1`.
