# ejecutor — Lanzador de cadenas de programas (lotes)

## Descripción

`ejecutor` lanza **cadenas de programas** conectadas por tuberías anónimas, emulando el comportamiento de un shell: `p1 | p2 | p3`. Lee los binarios directamente desde `aralmac/programas/` y registra el estado de cada lote en `aralmac/lotes/`, sin pasar por `gesfich` ni `gesprog`.

## Sinopsis

```
ejecutor -e <pipe-req> [-d <pipe-res>] -x <ruta_aralmac>
```

| Flag | Significado                                                     |
|------|-----------------------------------------------------------------|
| `-e` | Pipe de peticiones entrantes (desde ctrllt o cliente directo)   |
| `-d` | Pipe de respuestas salientes — solo en Linux (half‑duplex)      |
| `-x` | Ruta raíz del almacenamiento (`aralmac/`)                       |

## Operaciones

| Acción      | Descripción                                                                        | Retorna              |
|-------------|------------------------------------------------------------------------------------|----------------------|
| `ejecutar`  | Lanza la cadena de programas en background; retorna inmediatamente (no bloqueante) | `id_lote` (`l-XXXX`) |
| `estado`    | Devuelve el estado de un lote específico o la lista de todos los lotes             | JSON del lote / lista |
| `matar`     | Envía `SIGKILL` / `TerminateProcess` a **todos** los procesos hijos del lote       | `{ "id_lote", "estado": "matado", "message" }` |
| `suspender` | Suspende el servicio (rechaza nuevos lotes y `matar`)                              | `{ "estado": "Suspendido", "procesos_activos": N }` |
| `reasumir`  | Reanuda la aceptación de peticiones                                                | `{ "estado": "Corriendo", "procesos_activos": N }` |
| `parar`     | Cierre elegante: espera que terminen los lotes activos, luego sale                 | `{ "estado": "Parando", "procesos_activos": N }` |
| `terminar`  | Cierre inmediato: mata todos los procesos hijos y sale                             | `{ "estado": "Terminado", "procesos_activos": 0 }` |

### Payload de `ejecutar`

```json
{
  "id_fichero_entrada": "f-0001",
  "programas": ["p-0002", "p-0003", "p-0004"],
  "id_fichero_salida":  "f-0007"
}
```

- `programas`: array con uno o más IDs de programa en orden de ejecución.
- `id_fichero_entrada` e `id_fichero_salida` son obligatorios y deben existir en `aralmac`.
- Si cualquier programa o fichero no existe → `NOT_FOUND`. No se lanza ningún proceso.
- Incrementa `refcount` de los ficheros de entrada y salida antes de lanzar.
- Retorna `id_lote` inmediatamente; la terminación se detecta vía un hilo monitor.

## Estados del servicio

```
[Corriendo] ──suspender──> [Suspendido] ──reasumir──> [Corriendo]
    │                           │
    └────parar────> [Parando] ──(procesos_activos==0)──> [Terminado]
    │
    └────terminar─────────────────────────────────> [Terminado]
```

- **Corriendo:** acepta todas las operaciones.
- **Suspendido:** rechaza `ejecutar` y `matar` con `SERVICE_SUSPENDED`; acepta `estado`. Los lotes ya en ejecución continúan normalmente.
- **Parando:** rechaza `ejecutar` y `matar`; acepta `estado`. Termina automáticamente cuando no quedan lotes activos.
- **Terminado:** no acepta ninguna operación.

## Estados de un lote

```
corriendo → terminado
corriendo → fallido
corriendo → matado
```

| Estado      | Descripción                                                              |
|-------------|--------------------------------------------------------------------------|
| `corriendo` | Al menos un proceso hijo del lote sigue en ejecución                     |
| `terminado` | Todos los procesos hijos terminaron con código de salida 0               |
| `fallido`   | Algún proceso terminó con código de salida distinto de 0                 |
| `matado`    | El lote fue terminado forzosamente por la operación `matar`              |

## Gestión de la cadena de procesos

- **Linux:** `pipe()` para tuberías anónimas, `fork()` + `dup2()` + `execvp()` para cada programa.
- **Windows:** `CreatePipe()` para tuberías anónimas, `CreateProcess()` con `STARTUPINFO` para redirigir `hStdInput`/`hStdOutput`.
- Un hilo monitor espera con `waitpid` / `WaitForMultipleObjects` la terminación de todos los hijos; al terminar el último actualiza el estado del lote y decrementa `refcount` de los ficheros.
