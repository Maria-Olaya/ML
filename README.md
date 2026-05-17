# Ejecutor de Lotes

Sistema de ejecución por lotes al estilo mainframe. Los usuarios registran programas y ficheros en `aralmac` y solicitan la ejecución de lotes que conectan una **cadena de programas** mediante tuberías anónimas (`p1 | p2 | p3`).

**Integrantes:**
- Ana Sofia Alfonso Moncada
- Maria Mercedes Olaya Lopez

**Lenguaje:** Go
**Sistemas objetivo:** Linux y Windows 11
**Comunicación:** Named pipes — half‑duplex en Linux (2 pipes por conexión), full‑duplex en Windows (1 pipe por conexión)
**Protocolo:** JSON codificado en UTF-8, delimitado por `\n`

---

## Componentes

| Componente  | Descripción                                                          |
|-------------|----------------------------------------------------------------------|
| cliente     | Interfaz de usuario (proporcionada por el profesor). **Un único cliente simultáneo.** |
| ctrllt      | Pasarela que recibe peticiones del cliente y las enruta al servicio correspondiente |
| gesfich     | CRUD de ficheros de datos en `aralmac`                               |
| gesprog     | CRUD de programas ejecutables en `aralmac`                           |
| ejecutor    | Lanza y controla cadenas de programas (lotes)                        |
| aralmac     | Almacenamiento persistente en disco (directorio local)               |

---

## Arquitectura

```
cliente (único)
   |
   v
ctrllt
   |── ctrl-fich-req/res ──> gesfich ──> aralmac/ficheros/
   |── ctrl-prog-req/res ──> gesprog ──> aralmac/programas/
   └── ctrl-ejec-req/res ──> ejecutor ──> aralmac/lotes/
                                    └──> aralmac/ficheros/ (acceso directo)
                                    └──> aralmac/programas/ (acceso directo)
```

---

## Estructura del repositorio

```
Ejecutor_de_lotes/
├── documentos/
│   └── Diseño_v5.md
├── src/
│   ├── common/       # utilidades compartidas (JSON, pipes, locks, IDs, errores)
│   ├── ctrllt/       # pasarela central
│   ├── gesfich/      # CRUD ficheros
│   ├── gesprog/      # CRUD programas
│   └── ejecutor/     # lanzador de cadenas de programas
├── tests/
│   └── test_suite.sh
├── scripts/
│   ├── cleanup.sh    # limpieza en Linux
│   └── cleanup.ps1   # limpieza en Windows (PowerShell)
├── go.mod
└── README.md
```

---

## Protocolo de mensajes

Todos los mensajes son objetos JSON en UTF-8, una línea por mensaje terminada en `\n`.

**Request:**
```json
{
  "version":    "1.0",
  "client_id":  "cli-0001",
  "request_id": "req-000042",
  "service":    "gesfich",
  "action":     "crear",
  "payload":    {}
}
```

**Response:**
```json
{
  "version":    "1.0",
  "request_id": "req-000042",
  "status":     "ok",
  "data":       { "id_fichero": "f-0001" },
  "error":      null
}
```

---

## Tuberías nombradas

| Pipe            | Dirección              | Propósito                   |
|-----------------|------------------------|-----------------------------|
| `ctrl-cli`      | cliente ↔ ctrllt       | Comunicación con el cliente |
| `ctrl-fich-req` | ctrllt → gesfich       | Peticiones a gesfich        |
| `ctrl-fich-res` | gesfich → ctrllt       | Respuestas de gesfich       |
| `ctrl-prog-req` | ctrllt → gesprog       | Peticiones a gesprog        |
| `ctrl-prog-res` | gesprog → ctrllt       | Respuestas de gesprog       |
| `ctrl-ejec-req` | ctrllt → ejecutor      | Peticiones a ejecutor       |
| `ctrl-ejec-res` | ejecutor → ctrllt      | Respuestas de ejecutor      |

- Linux: `/tmp/lotes/<nombre_pipe>` (FIFOs, half‑duplex)
- Windows: `\\.\pipe\<nombre_pipe>` (full‑duplex, sin sufijos `-req`/`-res`)

---

## Orden de arranque

1. Levantar `gesfich`, `gesprog` y `ejecutor` (crean sus pipes y esperan).
2. Levantar `ctrllt` (conecta con todos los servicios).
3. Levantar el **único cliente** (conecta con `ctrllt` o directamente con un servicio).

### Terminación limpia
- El cliente envía `terminar` a `ctrllt`.
- `ctrllt` reenvía `terminar` a `gesfich`, `gesprog` y `ejecutor`, espera su confirmación y luego termina.

---

## Entregas

| Entrega  | Contenido                  | Fecha              |
|----------|----------------------------|--------------------|
| Primera  | `documentos/Diseño_v5.md`  | 5 de mayo de 2026  |
| Segunda  | Implementación completa    | Por definir        |

---

## Documentación

- [Diseño del sistema](documentos/Diseño_v5.md)
