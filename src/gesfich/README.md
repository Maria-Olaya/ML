# gesfich — Gestor de ficheros

## Descripción

`gesfich` administra ficheros de datos dentro del almacenamiento local (`aralmac/ficheros/`). Asigna identificadores únicos `f-XXXX`, controla el contador de referencias (`refcount`) y serializa el acceso concurrente mediante bloqueo exclusivo de archivos.

## Sinopsis

```
gesfich -f <pipe-req> [-b <pipe-res>] -x <ruta_aralmac>
```

| Flag | Significado                                                          |
|------|----------------------------------------------------------------------|
| `-f` | Pipe de peticiones entrantes (lo crea `gesfich` al arrancar)         |
| `-b` | Pipe de respuestas salientes — solo en Linux (half‑duplex)           |
| `-x` | Ruta raíz del almacenamiento (`aralmac/`)                            |

## Operaciones

| Acción      | Descripción                                                                   | Retorna                  |
|-------------|-------------------------------------------------------------------------------|--------------------------|
| `crear`     | Crea un fichero vacío en `aralmac/ficheros/` con `refcount = 0`               | `f-XXXX`                 |
| `leer`      | Devuelve el contenido en base64 (individual) o la lista de ficheros (listado) | JSON del fichero / lista |
| `actualizar`| Copia el contenido desde `ruta_origen` (ruta absoluta en el servidor) al `.dat` correspondiente | `{ "id_fichero", "size_bytes" }` |
| `borrar`    | Elimina el fichero solo si `refcount == 0`; si no → `RESOURCE_BUSY`          | `{ "id_fichero", "message": "eliminado" }` |
| `suspender` | Suspende el servicio                                                          | `{ "estado": "Suspendido" }` |
| `reasumir`  | Reanuda el servicio                                                           | `{ "estado": "Corriendo" }` |
| `terminar`  | Cierra pipes y termina el proceso                                             | `{ "estado": "Terminado" }` |

### Notas de operaciones

- **`crear`**: payload vacío `{}`. Crea `f-XXXX.dat` (vacío) y `f-XXXX.meta.json` con `{"size_bytes": 0, "refcount": 0}`.
- **`leer` individual**: payload `{ "id_fichero": "f-0001" }`. Devuelve el contenido codificado en base64.
- **`actualizar`**: payload `{ "id_fichero": "f-0001", "ruta_origen": "/ruta/absoluta" }`. Actualiza también `size_bytes` en el metadato.
- **`borrar`**: comprueba `refcount`; si `refcount > 0` responde `RESOURCE_BUSY`.

## Estados del proceso

```
Inicio → Corriendo ⇄ Suspendido → Terminado
```

- **Corriendo:** atiende todas las operaciones.
- **Suspendido:** **todas** las operaciones (incluyendo `leer`) responden con `SERVICE_SUSPENDED`. Solo se permite `reasumir` o `terminar`.
- **Terminado:** alcanzable desde `Corriendo` o `Suspendido`.

## Persistencia

```
aralmac/ficheros/
├── f-0001.dat           ← contenido binario
├── f-0001.meta.json     ← { "size_bytes": 1024, "refcount": 0 }
└── ...
```

El `refcount` lo incrementa `ejecutor` al iniciar un lote que usa el fichero, y lo decrementa al terminar (normal, fallido o matado).
