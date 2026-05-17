# ctrllt — Pasarela central

## Descripción

`ctrllt` es el punto de entrada para el **único cliente**. Recibe peticiones del cliente y las enruta al servicio correspondiente (`gesfich`, `gesprog` o `ejecutor`). Al haber un solo cliente, `ctrllt` opera de forma **secuencial** (un solo hilo): lee una petición, la reenvía al servicio, espera la respuesta y la devuelve al cliente.

Cuando recibe `action: "terminar"` dirigido a sí mismo, envía `terminar` a cada servicio, espera su confirmación y luego termina.

## Sinopsis

```
ctrllt -c <pipe-cli>      \
       -f <pipe-fich-req> [-b <pipe-fich-res>] \
       -p <pipe-prog-req> [-g <pipe-prog-res>] \
       -e <pipe-ejec-req> [-d <pipe-ejec-res>]
```

| Flag | Significado                                                          |
|------|----------------------------------------------------------------------|
| `-c` | Pipe de comunicación con el cliente                                  |
| `-f` | Pipe de peticiones hacia gesfich (`ctrl-fich-req`)                   |
| `-b` | Pipe de respuestas desde gesfich (`ctrl-fich-res`) — solo en Linux   |
| `-p` | Pipe de peticiones hacia gesprog (`ctrl-prog-req`)                   |
| `-g` | Pipe de respuestas desde gesprog (`ctrl-prog-res`) — solo en Linux   |
| `-e` | Pipe de peticiones hacia ejecutor (`ctrl-ejec-req`)                  |
| `-d` | Pipe de respuestas desde ejecutor (`ctrl-ejec-res`) — solo en Linux  |

En Windows los pipes son full‑duplex; en Linux se requieren dos pipes (req + res) por servicio.

## Responsabilidades

1. Leer un mensaje completo del cliente (pipe de petición).
2. Parsear el campo `service` del JSON para determinar el destino.
3. Reenviar el mensaje al pipe de petición del servicio correspondiente.
4. Esperar la respuesta leyendo del pipe de respuesta del servicio.
5. Escribir la respuesta en el pipe del cliente.
6. Repetir hasta recibir `terminar` o detectar pipe roto.

### Enrutamiento

| `service` en el JSON | Acción de ctrllt                                               |
|----------------------|----------------------------------------------------------------|
| `"gesfich"`          | Escribe en `ctrl-fich-req`, lee de `ctrl-fich-res`             |
| `"gesprog"`          | Escribe en `ctrl-prog-req`, lee de `ctrl-prog-res`             |
| `"ejecutor"`         | Escribe en `ctrl-ejec-req`, lee de `ctrl-ejec-res`             |
| `"ctrllt"`           | Procesa localmente (solo `terminar`)                           |
| otro                 | Responde `INVALID_ACTION`                                      |

## Estados del proceso

```
Inicio → Corriendo → Terminando → Terminado
```

- **Inicio:** abre y verifica todos los pipes antes de aceptar al cliente.
- **Corriendo:** acepta y despacha peticiones de forma secuencial.
- **Terminando:** recibió `terminar`; envía `terminar` a `gesfich`, `gesprog` y `ejecutor`, espera su `data.estado: "Terminado"` y luego cierra pipes.
- **Terminado:** proceso finalizado.

## Casos especiales

- **Pipe roto (cliente desaparecido):** `ctrllt` cierra sus extremos de pipe y termina (al haber un único cliente, no tiene sentido continuar).
- **Conexión directa del cliente a un servicio:** el cliente puede saltarse `ctrllt` y conectarse directamente a `gesfich`, `gesprog` o `ejecutor`. Cada servicio mantiene su propio bucle de recepción de peticiones y no depende exclusivamente de `ctrllt`.
