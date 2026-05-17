

# Diseño del Sistema – Ejecutor de Lotes

**Versión:** 5.0
**Fecha:** 2026-05-08

**Equipo:**
- Ana Sofia Alfonso Moncada
- Maria Mercedes Olaya Lopez

**Modalidad:** Grupo de dos integrantes
**Sistemas operativos objetivo:** Linux y Windows 11
**
---

## 1. Visión general del sistema

### Propósito
El sistema simula un entorno de ejecución por lotes (*batch processing*) al estilo de los mainframes. Los usuarios registran programas y ficheros en un área de almacenamiento central (`aralmac`) y luego solicitan la ejecución de lotes que asocian una **cadena de programas** con un fichero de entrada y un fichero de salida. La salida estándar de cada programa se redirige a la entrada estándar del siguiente (tuberías entre procesos), emulando el comportamiento de un shell: `programa1 | programa2 | programa3`.

### Componentes principales
- **cliente** – Interfaz de usuario (no implementado por el equipo, lo proporciona el profesor). **Solo existe un cliente** a la vez.
- **ctrllt** – Pasarela que recibe peticiones del cliente, las enruta al servicio correspondiente y devuelve las respuestas.
- **gesfich** – Gestor de ficheros: operaciones CRUD sobre ficheros almacenados en `aralmac`.
- **gesprog** – Gestor de programas: operaciones CRUD sobre programas (ejecutables, argumentos, variables de entorno).
- **ejecutor** – Lanza y controla lotes (cadenas de programas), utilizando los programas y ficheros registrados.
- **aralmac** – Área de almacenamiento persistente (directorio en disco).

### Limitaciones y alcance
- El sistema soporta **un único cliente** simultáneo (se confirmó que no se requieren múltiples clientes).
- Todos los componentes se comunican mediante tuberías nombradas (*named pipes*).
- **Duplexidad**:
  - En **Windows**: las tuberías son **full‑duplex** (una tubería bidireccional por conexión).
  - En **Linux**: las tuberías son **half‑duplex** (dos pipes unidireccionales por conexión).
- Los mensajes se codifican en JSON, delimitados por `\n`.
- No se implementan timeouts (mejora futura).
- El almacenamiento es persistente (los datos sobreviven a reinicios de los servicios).
- No se implementa streaming; el contenido de ficheros viaja en base64 dentro del JSON.

---

## 2. Estructura del repositorio

```
/
├── docs/
│   └── Diseño.md
├── src/
│   ├── ctrllt/
│   ├── gesfich/
│   ├── gesprog/
│   ├── ejecutor/
│   └── common/          ← utilidades compartidas (JSON, pipes, locks)
├── tests/
│   └── test_suite.sh    ← script de pruebas sugerido
├── scripts/
│   └── cleanup.sh       ← script de limpieza manual (borra aralmac y pipes)
└── README.md
```

---

## 3. Arquitectura de comunicación

### Tuberías nombradas (named pipes)
Todos los procesos utilizan *named pipes* (FIFO en Linux, *named pipe* en Windows) como único mecanismo de IPC.

### Full‑duplex en Windows, half‑duplex en Linux

| Sistema operativo | Modo | Pipes por conexión |
|------------------|------|--------------------|
| **Windows**      | Full‑duplex | 1 pipe bidireccional |
| **Linux**        | Half‑duplex | 2 pipes unidireccionales (petición + respuesta) |

Esto se implementa con archivos de compilación condicional por plataforma. Los nombres de las tuberías se adaptan según el caso.

### Nomenclatura de pipes (Linux / half‑duplex)

| Pipe            | Dirección            | Propósito                             |
|-----------------|----------------------|---------------------------------------|
| `ctrl-cli`      | cliente ↔ ctrllt     | Peticiones y respuestas (full‑duplex) / en Linux se usan dos pipes aparte |
| `ctrl-fich-req` | ctrllt → gesfich     | Peticiones a gesfich                  |
| `ctrl-fich-res` | gesfich → ctrllt     | Respuestas de gesfich                 |
| `ctrl-prog-req` | ctrllt → gesprog     | Peticiones a gesprog                  |
| `ctrl-prog-res` | gesprog → ctrllt     | Respuestas de gesprog                 |
| `ctrl-ejec-req` | ctrllt → ejecutor    | Peticiones a ejecutor                 |
| `ctrl-ejec-res` | ejecutor → ctrllt    | Respuestas de ejecutor                |

En **Windows** (full‑duplex) se usan los mismos nombres pero sin los sufijos `-req`/`-res`, ya que una sola tubería sirve para ambos sentidos.

**Ubicación en el sistema de archivos:**
- Linux: `/tmp/lotes/<nombre_pipe>`
- Windows: `\\.\pipe\<nombre_pipe>`

### El cliente puede comunicarse directamente con los servicios
Se indicó que el cliente **puede conectarse directamente** a `gesfich`, `gesprog` o `ejecutor`, además de hacerlo a través de `ctrllt`. Por lo tanto, cada servicio debe implementar su propio bucle de recepción de peticiones (a través de sus pipes) y no depender exclusivamente de `ctrllt`.

Sin embargo, para simplificar, en este diseño se asume que el cliente **siempre utilizará `ctrllt` como pasarela**, pero los servicios quedan preparados para aceptar conexiones directas si el usuario así lo requiere en su cliente.

---

## 4. Formato de mensajes JSON

Todos los mensajes son objetos JSON codificados en UTF-8. Cada mensaje termina con `\n` (newline). El receptor ignora `\r` si aparece.

### Estructura del Request

```json
{
  "version": "1.0",
  "client_id": "cli-0001",
  "request_id": "req-000001",
  "service": "gesfich | gesprog | ejecutor | ctrllt",
  "action": "nombre_operacion",
  "payload": { ... }
}
```

- `version`: fijo `"1.0"`.
- `client_id`: asignado por el cliente, patrón `cli-XXXX` (X dígito). **Como solo hay un cliente, este campo es informativo pero se mantiene por compatibilidad.**
- `request_id`: generado por el cliente, único por petición.
- `service`: servicio destino.
- `action`: operación a ejecutar.
- `payload`: objeto con parámetros específicos.

### Estructura del Response

```json
{
  "version": "1.0",
  "request_id": "req-000001",
  "status": "ok | error",
  "data": { ... },
  "error": null | { "code": "ERROR_CODE", "message": "texto descriptivo" }
}
```

- `request_id`: el mismo que el del request (para correlación).
- `status`: `"ok"` si éxito, `"error"` si fallo.
- `data`: presente solo si `status == "ok"`.
- `error`: presente solo si `status == "error"`.

### Códigos de error estándar

| Código               | Descripción                                                                  |
|----------------------|------------------------------------------------------------------------------|
| `NOT_FOUND`          | El recurso (fichero, programa, lote) no existe.                              |
| `ALREADY_EXISTS`     | El recurso ya existe (ej. intento de crear con ID duplicado).                |
| `INVALID_REQUEST`    | JSON mal formado, falta campo obligatorio, tipo incorrecto, o acción sobre recurso en estado inválido. |
| `INVALID_ACTION`     | La acción no está definida para el servicio.                                 |
| `SERVICE_SUSPENDED`  | El servicio está suspendido y no acepta esta operación.                      |
| `SERVICE_STOPPED`    | El servicio está en estado `Parando` o `Terminado`.                          |
| `INVALID_EXECUTABLE` | El archivo especificado no es ejecutable o no existe.                        |
| `EXECUTION_FAILED`   | Error al crear el proceso hijo.                                              |
| `RESOURCE_BUSY`      | El recurso está en uso (ej. borrar fichero referenciado por un lote activo). |
| `UNKNOWN_ERROR`      | Error interno no clasificado.                                                |

---

## 5. Identificadores del sistema

| Recurso   | Formato    | Ejemplo    | Generado por          |
|-----------|------------|------------|-----------------------|
| Fichero   | `f-XXXX`   | `f-0001`   | `gesfich`             |
| Programa  | `p-XXXX`   | `p-0001`   | `gesprog`             |
| Lote      | `l-XXXX`   | `l-0001`   | `ejecutor`            |
| Cliente   | `cli-XXXX` | `cli-0001` | Cliente (autogenerado)|

### Generación atómica de IDs
Cada servicio mantiene un archivo en `aralmac/secuencias/next_<tipo>.txt` con el último ID asignado. Para obtener un nuevo ID:
1. Abrir el archivo con bloqueo exclusivo (mecanismo nativo de cada SO: `flock` en Linux, `LockFile` en Windows, abstraído por el lenguaje).
2. Leer el número, incrementarlo, escribirlo de vuelta.
3. Liberar el bloqueo.
4. Formatear como `tipo-número` con 4 dígitos (ej. `f-0042`).
5. Si el archivo no existe, crearlo con valor inicial `1`.

Este mecanismo garantiza unicidad incluso bajo peticiones concurrentes y tras reinicios del servicio.

---

## 6. Servicio ctrllt (pasarela)

### Sinopsis
```
ctrllt -c <pipe-clientes> \
       -f <pipe-fich>    [-b <pipe-fich-res>] \
       -p <pipe-prog>    [-g <pipe-prog-res>] \
       -e <pipe-ejec>    [-d <pipe-ejec-res>]
```

- `-c`: pipe para comunicarse con el cliente (en Linux se usan dos pipes, pero el flag `-c` indica el pipe de petición; el de respuesta se pasa con `-a` o se ignora según la estrategia).  
  *En este diseño, al ser un solo cliente, se simplifica: `ctrllt` crea un pipe de petición (`ctrl-cli-req`) y el cliente crea su pipe de respuesta (`cli-res`), comunicando su nombre en el primer mensaje (aunque con un solo cliente no es estrictamente necesario).*
- `-f`, `-b`: pipes hacia y desde `gesfich` (en Linux se usan ambos; en Windows solo `-f`).
- `-p`, `-g`: pipes hacia y desde `gesprog`.
- `-e`, `-d`: pipes hacia y desde `ejecutor`.

### Máquina de estados

```
[Inicio] → [Corriendo] → [Terminando] → [Terminado]
```

- `Corriendo`: estado normal, acepta peticiones.
- `Terminando`: se ha solicitado `terminar`, se rechazan nuevas peticiones, se espera a que termine la petición actual (si la hay).
- `Terminado`: proceso finalizado.

### Concurrencia en ctrllt
Como **solo hay un cliente**, `ctrllt` puede ser **secuencial** (un solo hilo). No se necesitan hilos worker ni tabla de enrutamiento. El flujo es:

1. Leer un mensaje completo del cliente (desde el pipe de petición).
2. Parsear el JSON.
3. Reenviar el mensaje al servicio correspondiente (escribiendo en el pipe `*-req`).
4. Esperar la respuesta leyendo del pipe `*-res`.
5. Escribir la respuesta en el pipe de respuesta del cliente.
6. Repetir.

Si el cliente decide comunicarse directamente con los servicios, `ctrllt` no interviene.

### Reenvío según servicio
- `"gesfich"` → escribe en `ctrl-fich-req`, lee respuesta de `ctrl-fich-res`.
- `"gesprog"` → escribe en `ctrl-prog-req`, lee respuesta de `ctrl-prog-res`.
- `"ejecutor"` → escribe en `ctrl-ejec-req`, lee respuesta de `ctrl-ejec-res`.
- `"ctrllt"` → procesa localmente (solo `terminar`).
- Otro → responde `INVALID_ACTION`.

### Operación propia: `terminar`
**Cambio importante:** Cuando `ctrllt` recibe `action: "terminar"` dirigido a sí mismo, debe **terminar también los demás servicios** (`gesfich`, `gesprog` y `ejecutor`). Para ello:
1. Envía a cada servicio su propia orden `terminar` (a través de sus pipes de petición, con `action: "terminar"` y `payload: {}`).
2. Espera a que cada servicio confirme su terminación (respuesta con `status: "ok"` y `data.estado: "Terminado"`).
3. Una vez terminados todos, `ctrllt` procede a su propia terminación (cierra pipes y sale).

`ctrllt` no tiene operación de estado propia (solo enrutamiento y este `terminar`).

---

## 7. Servicio gesfich

### Sinopsis
```
gesfich -f <pipe-req> [-b <pipe-res>] -x <ruta_aralmac>
```
- `-f`: pipe de peticiones (lo crea `gesfich` al arrancar). En Windows es full‑duplex; en Linux se necesita también `-b` para respuestas.
- `-b`: pipe de respuestas (solo half‑duplex, Linux).
- `-x`: ruta al directorio `aralmac`.

### Máquina de estados

```
[Inicio] → [Corriendo] ⇄ [Suspendido] → [Terminado]
```

- En `Suspendido`: **todas** las operaciones (incluyendo `leer`) responden con error `SERVICE_SUSPENDED`.

### Operaciones

#### `crear`
- **Payload:** `{}`
- **Data éxito:** `{ "id_fichero": "f-0001" }`
- Crea `aralmac/ficheros/f-XXXX.dat` (vacío) y `f-XXXX.meta.json` con `{"size_bytes": 0, "refcount": 0}`.

#### `leer` (individual)
- **Payload:** `{ "id_fichero": "f-0001" }`
- **Data éxito:** `{ "id_fichero": "f-0001", "contenido": "<base64 del contenido>" }`
- Si no existe → `NOT_FOUND`.

#### `leer` (listado)
- **Payload:** `{}`
- **Data éxito:** `{ "ficheros": [ { "id_fichero": "f-0001", "size_bytes": 1024 }, ... ] }`

#### `actualizar`
- **Payload:** `{ "id_fichero": "f-0001", "ruta_origen": "/ruta/absoluta/en/servidor" }`
- **Data éxito:** `{ "id_fichero": "f-0001", "size_bytes": 2048 }`
- Copia el contenido del archivo local al `.dat` correspondiente. Actualiza tamaño en metadato.
- `ruta_origen` es una ruta absoluta en el sistema de archivos del servidor.
- Errores: `NOT_FOUND` (id o ruta inexistentes), `SERVICE_SUSPENDED`.

#### `borrar`
- **Payload:** `{ "id_fichero": "f-0001" }`
- **Data éxito:** `{ "id_fichero": "f-0001", "message": "eliminado" }`
- Comprueba `refcount`. Si `refcount > 0` → `RESOURCE_BUSY`. Si no, elimina `.dat` y `.meta.json`.

#### `suspender` / `reasumir` / `terminar`
- **Payload:** `{}`
- **Data éxito:** `{ "estado": "Suspendido" | "Corriendo" | "Terminado" }`

### Persistencia
- `aralmac/ficheros/f-XXXX.dat` — contenido binario del fichero.
- `aralmac/ficheros/f-XXXX.meta.json`:
```json
{ "size_bytes": 1024, "refcount": 0 }
```
- `refcount` lo incrementa `ejecutor` al iniciar un lote que usa el fichero, y lo decrementa al terminar.

---

## 8. Servicio gesprog

### Sinopsis
```
gesprog -p <pipe-req> [-g <pipe-res>] -x <ruta_aralmac>
```

### Máquina de estados
Idéntica a `gesfich`, **excepto** que en estado `Suspendido` la operación `leer` **sí está permitida** (el resto devuelven `SERVICE_SUSPENDED`).

### Operaciones

#### `guardar`
- **Payload:**
```json
{
  "ruta_ejecutable": "/bin/mi_programa",
  "argumentos": ["--opt", "valor"],
  "ambiente": { "PATH": "/bin", "LANG": "es" }
}
```
- **Data éxito:** `{ "id_programa": "p-0001" }`
- Valida que `ruta_ejecutable` exista y sea ejecutable. Copia el binario a `aralmac/programas/p-XXXX.bin`. Guarda metadatos en `p-XXXX.meta.json`.
- Errores: `INVALID_EXECUTABLE`, `SERVICE_SUSPENDED`.

#### `leer` (individual)
- **Payload:** `{ "id_programa": "p-0001" }`
- **Data éxito:**
```json
{
  "id_programa": "p-0001",
  "ruta_ejecutable": "/bin/mi_programa",
  "argumentos": ["--opt", "valor"],
  "ambiente": { "PATH": "/bin", "LANG": "es" }
}
```
- `ruta_ejecutable` es la ruta original informativa, no la ruta dentro de `aralmac`.

#### `leer` (listado)
- **Payload:** `{}`
- **Data éxito:** `{ "programas": [ { ... }, ... ] }`

#### `actualizar`
- **Payload:** `{ "id_programa": "p-0001", "ruta_origen": "/nuevo/ejecutable" }`
- **Data éxito:** `{ "id_programa": "p-0001", "message": "actualizado" }`
- Reemplaza solo el binario. Argumentos y ambiente se mantienen sin cambio.

#### `borrar`
- **Payload:** `{ "id_programa": "p-0001" }`
- **Data éxito:** `{ "id_programa": "p-0001", "message": "eliminado" }`
- Elimina `.bin` y `.meta.json`.

#### `suspender` / `reasumir` / `terminar`
- Igual que en `gesfich`.

### Almacenamiento
- `aralmac/programas/p-XXXX.bin`
- `aralmac/programas/p-XXXX.meta.json`:
```json
{
  "ruta_original": "/bin/mi_programa",
  "argumentos": ["--opt", "valor"],
  "ambiente": { "PATH": "/bin", "LANG": "es" }
}
```

---

## 9. Servicio ejecutor

### Sinopsis
```
ejecutor -e <pipe-req> [-d <pipe-res>] -x <ruta_aralmac>
```

### Máquina de estados

```
[Corriendo] ──suspender──> [Suspendido] ──reasumir──> [Corriendo]
    │                           │
    └────parar────> [Parando] ──(procesos_activos==0)──> [Terminado]
    │
    └────terminar─────────────────────────────────────> [Terminado]
```

- **Corriendo:** acepta todas las operaciones.
- **Suspendido:** rechaza `ejecutar` y `matar` con `SERVICE_SUSPENDED`; acepta `estado`. Los lotes ya en ejecución continúan normalmente.
- **Parando:** rechaza `ejecutar` y `matar`; acepta `estado`. Al llegar a 0 lotes activos, el servicio termina automáticamente.
- **Terminado:** no acepta ninguna operación.

### Operaciones

#### `ejecutar` – Lanzar una cadena de programas (lote)
**Cambio fundamental:** El payload ahora permite especificar **múltiples programas** que se ejecutarán en serie, conectando la salida de cada uno con la entrada del siguiente. El formato es:

```json
{
  "id_fichero_entrada": "f-0001",      // obligatorio
  "programas": ["p-0002", "p-0003", "p-0004"],  // al menos un programa
  "id_fichero_salida": "f-0007"        // obligatorio
}
```

- `id_fichero_entrada` y `id_fichero_salida` son **obligatorios** en el JSON y deben existir previamente en `aralmac`. Si alguno no existe → `NOT_FOUND`.
- `programas` es un array con uno o más identificadores de programa (orden de ejecución). Es **obligatorio** y debe tener al menos un elemento.
- El comportamiento interno:
  1. Verificar que todos los programas existan en `aralmac`. Si alguno no existe → `NOT_FOUND`.
  2. Verificar que `id_fichero_entrada` e `id_fichero_salida` existan en `aralmac`. Si alguno no existe → `NOT_FOUND`.
  3. Incrementar `refcount` de los ficheros de entrada y salida.
  3. Crear una **tubería anónima** (pipe) por cada par de programas consecutivos.
  4. Para cada programa, lanzar un proceso hijo:
     - El primer programa recibe su stdin del fichero de entrada (o de `/dev/null` si no se pudo crear).
     - Los programas intermedios reciben su stdin del extremo de lectura de la tubería anterior.
     - Los programas intermedios envían su stdout al extremo de escritura de la siguiente tubería.
     - El último programa envía su stdout al fichero de salida.
  5. Guardar los PIDs de todos los procesos hijos y el estado del lote en `aralmac/lotes/l-XXXX.json`.
  6. Retornar `id_lote` inmediatamente (no bloqueante).

- **Un solo lote** agrupa toda la cadena; cuando todos los procesos hijos hayan terminado (o uno falle), se actualiza el estado global del lote (`terminado`, `fallido`).

- **Data éxito (inmediato):**
```json
{
  "id_lote": "l-0001",
  "estado": "corriendo",
  "programas": ["p-0002", "p-0003", "p-0004"],
  "timestamp_inicio": "2026-05-07T10:00:00Z"
}
```

#### `estado` (individual)
- **Payload:** `{ "id_lote": "l-0001" }`
- **Data éxito:**
```json
{
  "id_lote": "l-0001",
  "estado": "corriendo | terminado | fallido | matado",
  "codigo_salida": 0,   // código del último programa o del que falló
  "programas": ["p-0002", "p-0003", "p-0004"],
  "timestamp_inicio": "...",
  "timestamp_fin": "..."
}
```

#### `estado` (listado)
- **Payload:** `{}`
- **Data éxito:** `{ "procesos": [ { ... }, ... ] }`

#### `matar`
- **Payload:** `{ "id_lote": "l-0001" }`
- **Data éxito:** `{ "id_lote": "l-0001", "estado": "matado", "message": "terminado forzosamente" }`
- Envía una señal de terminación forzosa a **todos** los procesos hijos del lote (equivalente a `SIGKILL` en Linux y `TerminateProcess` en Windows). Actualiza estado y decrementa `refcount` de los ficheros.
- Si el lote ya no está corriendo → `INVALID_REQUEST`.

#### `suspender` / `reasumir` / `parar` / `terminar`
- **Payload:** `{}`
- **Data ejemplo (suspender):** `{ "estado": "Suspendido", "procesos_activos": 2 }`
- `parar`: apagado elegante, espera que los lotes activos terminen solos.
- `terminar`: mata todos los hijos inmediatamente y sale.

### Acceso directo a aralmac
`ejecutor` no se comunica con `gesfich` ni `gesprog` a través de sus pipes. Lee y escribe directamente en `aralmac` para evitar ineficiencias y posibles deadlocks. El acceso concurrente a los archivos `.meta.json` se protege con bloqueo exclusivo de archivo.

### Gestión de procesos hijos (cadena)
En ambas plataformas se utilizan las primitivas del sistema operativo para:
- Crear tuberías anónimas entre procesos consecutivos de la cadena (en Linux: `pipe` + `dup2`; en Windows: `CreatePipe` con redirección de manejadores).
- Lanzar cada proceso hijo con su stdin/stdout redirigido al pipe correspondiente (en Linux: `fork` + `execvp`; en Windows: `CreateProcess`).
- Un hilo monitor por lote espera la terminación de todos los procesos hijos (en Linux: `waitpid`; en Windows: `WaitForMultipleObjects`). Al terminar el último, actualiza el estado global del lote y decrementa `refcount`.

El lenguaje elegido (Go) abstrae estas diferencias en su biblioteca estándar (`os/exec`, `os.Pipe`), pero el comportamiento es equivalente al descrito.

---

## 10. Almacenamiento (aralmac)

### Estructura de directorios
```
aralmac/
├── ficheros/
│   ├── f-0001.dat
│   ├── f-0001.meta.json
│   └── ...
├── programas/
│   ├── p-0001.bin
│   ├── p-0001.meta.json
│   └── ...
├── lotes/
│   ├── l-0001.json
│   └── ...
└── secuencias/
    ├── next_fichero.txt
    ├── next_programa.txt
    └── next_lote.txt
```

### Formato de metadatos

**Fichero (`f-XXXX.meta.json`):**
```json
{ "size_bytes": 1024, "refcount": 0 }
```

**Programa (`p-XXXX.meta.json`):**
```json
{
  "ruta_original": "/bin/mi_programa",
  "argumentos": ["--opt", "valor"],
  "ambiente": { "PATH": "/bin", "LANG": "es" }
}
```

**Lote (`l-XXXX.json`):**
```json
{
  "programas": ["p-0002", "p-0003", "p-0004"],
  "pids": [12345, 12346, 12347],
  "estado": "corriendo",
  "id_fichero_entrada": "f-0001",
  "id_fichero_salida": "f-0007",
  "timestamp_inicio": "2026-05-07T10:00:00Z",
  "timestamp_fin": null,
  "codigo_salida": null
}
```

### Persistencia
- Todos los datos se escriben en disco inmediatamente tras cada cambio.
- Al reiniciar, cada servicio escanea su directorio para reconstruir el estado interno.

### Script de limpieza manual

Se proporcionan dos scripts según el sistema operativo:

- **Linux:** `scripts/cleanup.sh`
- **Windows:** `scripts/cleanup.ps1` (PowerShell)

Estos scripts **no se ejecutan automáticamente**. El usuario debe lanzarlos manualmente cuando desee resetear el sistema, por ejemplo antes de una nueva tanda de pruebas o tras una terminación anómala que deje pipes o archivos huérfanos.

#### `scripts/cleanup.sh` (Linux)

```bash
#!/bin/bash
# Limpia el entorno del Ejecutor de Lotes en Linux.
# Uso:
#   bash scripts/cleanup.sh          → limpieza rápida (pipes + estado de lotes)
#   bash scripts/cleanup.sh --full   → limpieza completa (también ficheros y programas)

ARALMAC=${ARALMAC:-"./aralmac"}
PIPES_DIR="/tmp/lotes"
FULL=false
for arg in "$@"; do [ "$arg" = "--full" ] && FULL=true; done

echo "Limpiando entorno..."

# 1. Eliminar named pipes (FIFOs)
if [ -d "$PIPES_DIR" ]; then
  rm -f "$PIPES_DIR"/ctrl-cli-req "$PIPES_DIR"/ctrl-cli-res
  rm -f "$PIPES_DIR"/ctrl-fich-req "$PIPES_DIR"/ctrl-fich-res
  rm -f "$PIPES_DIR"/ctrl-prog-req "$PIPES_DIR"/ctrl-prog-res
  rm -f "$PIPES_DIR"/ctrl-ejec-req "$PIPES_DIR"/ctrl-ejec-res
  rmdir "$PIPES_DIR" 2>/dev/null || true
  echo "  -> Pipes eliminados."
fi

# 2. Eliminar estado de lotes
rm -f "$ARALMAC"/lotes/*.json
echo "  -> Estado de lotes eliminado."

# 3. Limpieza completa (opcional)
if [ "$FULL" = true ]; then
  rm -f "$ARALMAC"/ficheros/*.dat "$ARALMAC"/ficheros/*.meta.json
  rm -f "$ARALMAC"/programas/*.bin "$ARALMAC"/programas/*.meta.json
  rm -f "$ARALMAC"/secuencias/next_*.txt
  echo "  -> Ficheros y programas eliminados (aralmac limpio)."
fi

echo "Listo. El sistema puede arrancarse de nuevo."
```

#### `scripts/cleanup.ps1` (Windows / PowerShell)

```powershell
# Limpia el entorno del Ejecutor de Lotes en Windows.
# Uso:
#   .\scripts\cleanup.ps1          -> limpieza rapida (estado de lotes)
#   .\scripts\cleanup.ps1 -Full    -> limpieza completa (tambien ficheros y programas)
# Nota: los named pipes full-duplex de Windows se cierran solos al terminar
# el proceso servidor; no es necesario borrarlos manualmente.

param([switch]$Full)
$Aralmac = if ($env:ARALMAC) { $env:ARALMAC } else { ".\aralmac" }

Write-Host "Limpiando entorno..." -ForegroundColor Cyan

# 1. Eliminar estado de lotes
Remove-Item "$Aralmac\lotes\*.json" -Force -ErrorAction SilentlyContinue
Write-Host "  -> Estado de lotes eliminado."

# 2. Limpieza completa (opcional)
if ($Full) {
  Remove-Item "$Aralmac\ficheros\*.dat"        -Force -ErrorAction SilentlyContinue
  Remove-Item "$Aralmac\ficheros\*.meta.json"  -Force -ErrorAction SilentlyContinue
  Remove-Item "$Aralmac\programas\*.bin"       -Force -ErrorAction SilentlyContinue
  Remove-Item "$Aralmac\programas\*.meta.json" -Force -ErrorAction SilentlyContinue
  Remove-Item "$Aralmac\secuencias\next_*.txt" -Force -ErrorAction SilentlyContinue
  Write-Host "  -> Ficheros y programas eliminados (aralmac limpio)."
}

Write-Host "Listo. El sistema puede arrancarse de nuevo." -ForegroundColor Green
```

#### Qué borra cada modo

| Elemento | Limpieza rápida | Limpieza completa (`--full` / `-Full`) |
|---|---|---|
| Named pipes (Linux FIFOs en `/tmp/lotes/`) | ✅ | ✅ |
| Named pipes (Windows) | automático al cerrar servicios | automático al cerrar servicios |
| Estado de lotes (`aralmac/lotes/*.json`) | ✅ | ✅ |
| Ficheros de datos (`aralmac/ficheros/`) | ❌ | ✅ |
| Programas registrados (`aralmac/programas/`) | ❌ | ✅ |
| Contadores de secuencia (`aralmac/secuencias/`) | ❌ | ✅ |

### Contador de referencias (`refcount`)
- `gesfich` inicializa `refcount = 0` al crear un fichero.
- `ejecutor` incrementa `refcount` antes de lanzar un lote y lo decrementa al terminar (normal, fallido o matado).
- El acceso a `refcount` se protege con bloqueo exclusivo de archivo para evitar condiciones de carrera entre procesos concurrentes.
- Si `gesfich` recibe `borrar` y `refcount > 0` → `RESOURCE_BUSY`.

---

## 11. Concurrencia y sincronización

### Modelo de hilos
- **`ctrllt`**: al ser solo un cliente, es secuencial (un solo hilo). No se usan hilos worker.
- **`gesfich` y `gesprog`**: pueden ser secuenciales (un hilo principal atiende las peticiones una a una) o usar hilos worker; al haber un solo cliente no es crítica la concurrencia.
- **`ejecutor`**: sí requiere concurrencia interna: un hilo principal atiende peticiones (`ejecutar`, `estado`, `matar`, etc.) y por cada lote se lanzan múltiples procesos hijos (uno por programa en la cadena). Además, un hilo monitor se encarga de recolectar procesos terminados.

### Protección de estructuras compartidas
- Lista de lotes en `ejecutor`: protegida con mutex.
- Archivos de secuencia en `aralmac`: bloqueo exclusivo a nivel de archivo (mecanismo nativo del SO, abstraído por el lenguaje).
- Archivos `.meta.json` (para `refcount`): bloqueo exclusivo a nivel de archivo.

### Evitar deadlocks
- No se adquieren múltiples locks anidados que puedan causar ciclos.
- `ejecutor` no mantiene ningún lock mientras accede a `aralmac`; cada operación abre, bloquea, modifica, desbloquea y cierra.

---

## 12. Manejo de errores y casos especiales

| Situación | Comportamiento |
|---|---|
| JSON mal formado | Responde `INVALID_REQUEST`, continúa escuchando |
| Pipe roto (cliente desaparecido) | `ctrllt` cierra sus extremos y termina (ya que solo hay un cliente) |
| Pipe roto hacia `ctrllt` | El servicio afectado finaliza |
| Fallo al lanzar algún proceso de la cadena | Responde `EXECUTION_FAILED` y mata los procesos ya lanzados de ese lote |
| Matar lote ya terminado | Responde `INVALID_REQUEST` |
| Borrar fichero con refcount > 0 | Responde `RESOURCE_BUSY` |
| Servicio suspendido | Responde `SERVICE_SUSPENDED` sin procesar |
| Ejecutable inválido en guardar/actualizar | Responde `INVALID_EXECUTABLE` |
| Acción desconocida | Responde `INVALID_ACTION` |
| Fichero de entrada o salida no existe en `ejecutar` | Responde `NOT_FOUND`. No se lanza ningún proceso. |

---

## 13. Orden de arranque y terminación

### Arranque (orden recomendado)
1. `gesfich`, `gesprog`, `ejecutor` (en cualquier orden) — crean sus pipes.
2. `ctrllt` — abre los pipes de los servicios, crea su pipe de comunicación con el cliente.
3. El **único cliente** — crea su pipe de respuesta (si no usa full‑duplex) y envía su primer mensaje a `ctrllt` (o directamente a un servicio).

### Terminación limpia
1. El cliente cierra su pipe (o termina).
2. Se envía `terminar` a `ctrllt`.
3. `ctrllt` envía `terminar` a `gesfich`, `gesprog` y `ejecutor`, espera su finalización y luego termina.

---

## 14. Consideraciones por sistema operativo

| Aspecto | Linux | Windows |
|---|---|---|
| Tipo de pipe nombrado | FIFO half-duplex | Named Pipe full-duplex |
| Creación de pipe nombrado | `mkfifo` | `CreateNamedPipe` con `PIPE_ACCESS_DUPLEX` |
| Apertura de pipe | `open` con flags de lectura/escritura | `CreateFile` con permisos de lectura/escritura |
| Bloqueo exclusivo de archivo | `flock` | `LockFile` / `UnlockFile` |
| Hilos | Hilos POSIX (`pthread`) | Hilos Win32 (`CreateThread`) |
| Mutex | `pthread_mutex_lock` | `WaitForSingleObject` |
| Matar proceso hijo | Señal `SIGKILL` | `TerminateProcess` |
| Redirección stdin/stdout | `dup2` | `STARTUPINFO` con manejadores |
| Tubería anónima (cadena) | `pipe` | `CreatePipe` |
| Lanzar proceso hijo | `fork` + `execvp` | `CreateProcess` |
| Esperar proceso hijo | `waitpid` | `WaitForSingleObject` / `WaitForMultipleObjects` |

Go abstrae todas estas diferencias en su biblioteca estándar. El código condicional por plataforma se limita a la creación y apertura de named pipes; el resto (lanzar procesos, redirigir stdin/stdout, crear pipes anónimos, esperar hijos) es idéntico en ambas plataformas gracias al paquete `os/exec`.

---

## 15. Flujos de ejemplo end-to-end

### Registro de un programa
1. Cliente envía a `ctrl-cli` (o directamente a `gesprog`):
```json
{ "version":"1.0", "client_id":"cli-0001", "request_id":"r1", "service":"gesprog", "action":"guardar", "payload":{ "ruta_ejecutable":"/bin/echo", "argumentos":["hola"], "ambiente":{} } }
```
2. `gesprog` valida, copia a `aralmac/programas/p-0001.bin`, guarda metadatos, responde.

### Creación y actualización de un fichero
1. Cliente envía `crear` a `gesfich` → recibe `f-0001`.
2. Cliente envía `actualizar` con `ruta_origen` apuntando a un archivo local en el servidor.

### Ejecución de un lote con cadena de programas
1. Cliente envía a `ejecutor`:
```json
{
  "action": "ejecutar",
  "payload": {
    "id_fichero_entrada": "f-0001",
    "programas": ["p-0002", "p-0003", "p-0004"],
    "id_fichero_salida": "f-0007"
  }
}
```
2. `ejecutor` verifica que todos los programas y ambos ficheros existan en `aralmac`. Si alguno no existe → `NOT_FOUND`. Incrementa `refcount` de `f-0001` y `f-0007`.
3. Lanza tres procesos hijos conectados por tuberías anónimas: `p-0002 | p-0003 | p-0004`, con stdin desde `f-0001` y stdout hacia `f-0007`.
4. Retorna `id_lote = l-0001` inmediatamente.
5. Cliente consulta `estado` periódicamente hasta que vea `terminado` o `fallido`.
6. Al terminar todos los hijos, `ejecutor` actualiza estado y decrementa `refcount`.

### Suspensión y reanudación de un servicio
1. Cliente envía `suspender` a `gesfich`.
2. `gesfich` cambia a `Suspendido`.
3. Cualquier otra petición (excepto `reasumir` o `terminar`) recibe `SERVICE_SUSPENDED`.
4. Cliente envía `reasumir` → vuelve a `Corriendo`.

---

## 16. Pruebas y validación

### Casos de prueba mínimos

**Happy path**
1. Registrar tres programas (`p-0001`, `p-0002`, `p-0003`).
2. Crear fichero de entrada `f-0001` con contenido.
3. Crear fichero de salida vacío `f-0002`.
4. Ejecutar lote con cadena: `f-0001` → `p-0001` → `p-0002` → `p-0003` → `f-0002`.
5. Consultar estado hasta que termine.
6. Leer `f-0002` y verificar contenido (es la salida del último programa).

**Errores**
- Leer fichero inexistente → `NOT_FOUND`.
- Guardar programa con ejecutable no válido → `INVALID_EXECUTABLE`.
- Borrar fichero mientras un lote lo usa → `RESOURCE_BUSY`.
- Enviar JSON mal formado → `INVALID_REQUEST`.
- Enviar acción desconocida → `INVALID_ACTION`.
- `ejecutar` con un programa que no existe → `NOT_FOUND`.

**Cadena de programas**
- Probar con 1 solo programa (debe funcionar igual).
- Probar con 3 programas donde el segundo falla → el lote debe marcarse como `fallido` y los procesos restantes deben terminarse.

**Estados suspendido**
- `gesfich` suspendido → cualquier operación (incluyendo `leer`) da error.
- `gesprog` suspendido → `leer` funciona, `guardar` da error.
- `ejecutor` suspendido → `ejecutar` da error, `estado` funciona.

**Terminación**
- `ctrllt terminar` debe terminar todos los servicios y luego salir.
- `ejecutor parar` debe esperar a que terminen los lotes activos y luego salir.

### Script de pruebas sugerido
Se proporciona `tests/test_suite.sh` (y su equivalente `.bat`) que lanza los servicios, ejecuta una secuencia de comandos mediante un cliente de prueba y verifica las respuestas. El script de limpieza manual (`scripts/cleanup.sh`) se puede ejecutar antes de cada ejecución de pruebas si se desea un estado completamente limpio.

### Criterios de aceptación
- El sistema pasa todos los casos happy path.
- Responde con los códigos de error definidos ante entradas incorrectas.
- No se producen caídas ni fugas de recursos (pipes no cerrados, procesos zombies).
- La cadena de programas se ejecuta correctamente, respetando la redirección de entrada/salida entre ellos.

---

## 17. Apéndice: Schemas JSON completos

### Schemas base

```json
{
  "Request": {
    "type": "object",
    "required": ["version", "client_id", "request_id", "service", "action", "payload"],
    "properties": {
      "version":    { "type": "string", "enum": ["1.0"] },
      "client_id":  { "type": "string", "pattern": "^cli-[0-9]{4}$" },
      "request_id": { "type": "string" },
      "service":    { "type": "string", "enum": ["ctrllt", "gesfich", "gesprog", "ejecutor"] },
      "action":     { "type": "string" },
      "payload":    { "type": "object" }
    }
  },
  "Response": {
    "type": "object",
    "required": ["version", "request_id", "status", "data", "error"],
    "properties": {
      "version":    { "type": "string", "enum": ["1.0"] },
      "request_id": { "type": "string" },
      "status":     { "type": "string", "enum": ["ok", "error"] },
      "data":       { "type": ["object", "null"] },
      "error": {
        "oneOf": [
          { "type": "null" },
          {
            "type": "object",
            "required": ["code", "message"],
            "properties": {
              "code":    { "type": "string" },
              "message": { "type": "string" }
            }
          }
        ]
      }
    }
  }
}
```

### Payloads específicos

**gesfich – crear**
```json
{ "payload": {} }
// éxito: { "id_fichero": "f-0001" }
```

**gesfich – leer (individual)**
```json
{ "payload": { "id_fichero": "f-0001" } }
// éxito: { "id_fichero": "f-0001", "contenido": "base64..." }
```

**gesfich – leer (listado)**
```json
{ "payload": {} }
// éxito: { "ficheros": [ { "id_fichero": "f-0001", "size_bytes": 1024 } ] }
```

**gesfich – actualizar**
```json
{ "payload": { "id_fichero": "f-0001", "ruta_origen": "/ruta/absoluta" } }
// éxito: { "id_fichero": "f-0001", "size_bytes": 2048 }
```

**gesfich – borrar**
```json
{ "payload": { "id_fichero": "f-0001" } }
// éxito: { "id_fichero": "f-0001", "message": "eliminado" }
```

**gesprog – guardar**
```json
{ "payload": { "ruta_ejecutable": "/bin/prog", "argumentos": ["--opt"], "ambiente": { "VAR": "valor" } } }
// éxito: { "id_programa": "p-0001" }
```

**gesprog – leer (individual)**
```json
{ "payload": { "id_programa": "p-0001" } }
// éxito: { "id_programa": "p-0001", "ruta_ejecutable": "...", "argumentos": [...], "ambiente": {...} }
```

**gesprog – leer (listado)**
```json
{ "payload": {} }
// éxito: { "programas": [ { ... } ] }
```

**gesprog – actualizar**// éxito: { "id_lote": "l-0001", "estado": "corriendo", "timestamp_inicio": "..." }


-a
 
	This is an evaluation version of Markdown Monster. For continued use, please register this copy.
```json
{ "payload": { "id_programa": "p-0001", "ruta_origen": "/nuevo/ejecutable" } }
// éxito: { "id_programa": "p-0001", "message": "actualizado" }
```

**gesprog – borrar**
```json
{ "payload": { "id_programa": "p-0001" } }
// éxito: { "id_programa": "p-0001", "message": "eliminado" }
```

**ejecutor – ejecutar (nuevo formato)**
```json
{
  "payload": {
    "id_fichero_entrada": "f-0001",
    "programas": ["p-0002", "p-0003", "p-0004"],
    "id_fichero_salida": "f-0007"
  }
}
// éxito: { "id_lote": "l-0001", "estado": "corriendo", "timestamp_inicio": "..." }
```

**ejecutor – estado (individual)**
```json
{ "payload": { "id_lote": "l-0001" } }
// éxito: { "id_lote": "l-0001", "estado": "...", "codigo_salida": 0, "timestamp_inicio": "...", "timestamp_fin": "..." }
```

**ejecutor – estado (listado)**
```json
{ "payload": {} }
// éxito: { "procesos": [ ... ] }
```

**ejecutor – matar**
```json
{ "payload": { "id_lote": "l-0001" } }
// éxito: { "id_lote": "l-0001", "estado": "matado", "message": "terminado forzosamente" }
```

**suspender / reasumir / terminar / parar**
```json
{ "payload": {} }
// éxito: { "estado": "Suspendido | Corriendo | Terminado | Parando", "procesos_activos": N }
```

# Maquinas de estados.

<img width="427" height="165" alt="image" src="https://github.com/user-attachments/assets/642b86bf-e877-461f-bb4e-03059df54cca" />
---
<img width="568" height="385" alt="image" src="https://github.com/user-attachments/assets/5ff2da16-066c-4c6d-8ec9-3b26b8a013e9" />
---
<img width="602" height="386" alt="image" src="https://github.com/user-attachments/assets/1ea55c20-8f08-4dd7-b377-c2e45b68ebf5" />
---
<img width="602" height="447" alt="image" src="https://github.com/user-attachments/assets/ee2e7112-58e3-4eb3-a801-2be9f7809358" />


---
