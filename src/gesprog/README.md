# gesprog — Gestor de programas

## Descripción

`gesprog` administra los binarios ejecutables dentro del almacenamiento local (`aralmac/programas/`). Asigna identificadores únicos `p-XXXX` y controla el acceso concurrente mediante bloqueo exclusivo de archivos.

## Sinopsis

```
gesprog -p <pipe-req> [-g <pipe-res>] -x <ruta_aralmac>
```

| Flag | Significado                                                          |
|------|----------------------------------------------------------------------|
| `-p` | Pipe de peticiones entrantes (lo crea `gesprog` al arrancar)         |
| `-g` | Pipe de respuestas salientes — solo en Linux (half‑duplex)           |
| `-x` | Ruta raíz del almacenamiento (`aralmac/`)                            |

## Operaciones

| Acción      | Descripción                                                                         | Retorna                   |
|-------------|-------------------------------------------------------------------------------------|---------------------------|
| `guardar`   | Valida que el ejecutable exista, copia el binario al almacén, guarda metadatos      | `p-XXXX`                  |
| `leer`      | Devuelve metadatos de un programa (individual) o la lista completa                  | JSON del programa / lista |
| `actualizar`| Reemplaza solo el binario; argumentos y ambiente se mantienen sin cambio            | `{ "id_programa", "message": "actualizado" }` |
| `borrar`    | Elimina el binario y sus metadatos del almacén                                      | `{ "id_programa", "message": "eliminado" }` |
| `suspender` | Suspende el servicio                                                                | `{ "estado": "Suspendido" }` |
| `reasumir`  | Reanuda el servicio                                                                 | `{ "estado": "Corriendo" }` |
| `terminar`  | Cierra pipes y termina el proceso                                                   | `{ "estado": "Terminado" }` |

### Notas de operaciones

- **`guardar`**: valida que `ruta_ejecutable` exista y sea ejecutable en el sistema; si no → `INVALID_EXECUTABLE`. Payload:
  ```json
  { "ruta_ejecutable": "/bin/prog", "argumentos": ["--opt"], "ambiente": { "VAR": "valor" } }
  ```
- **`leer` individual**: payload `{ "id_programa": "p-0001" }`. Devuelve `ruta_original`, `argumentos` y `ambiente`.
- **`actualizar`**: payload `{ "id_programa": "p-0001", "ruta_origen": "/nuevo/ejecutable" }`. Solo reemplaza el binario.

## Estados del proceso

```
Inicio → Corriendo ⇄ Suspendido → Terminado
```

- **Corriendo:** atiende todas las operaciones.
- **Suspendido:** la operación `leer` **sí está permitida**; las operaciones de escritura (`guardar`, `actualizar`, `borrar`) responden con `SERVICE_SUSPENDED`. Solo estas dos excepciones; `terminar` también se permite.
- **Terminado:** alcanzable desde `Corriendo` o `Suspendido`.

## Persistencia

```
aralmac/programas/
├── p-0001.bin           ← binario copiado del ejecutable original
├── p-0001.meta.json     ← { "ruta_original": "...", "argumentos": [...], "ambiente": {...} }
└── ...
```
