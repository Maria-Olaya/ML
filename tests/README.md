# tests — Plan de pruebas

Las pruebas se organizan por categoría. El script principal es `tests/test_suite.sh` (Linux) y su equivalente `tests/test_suite.bat` (Windows), que lanzan los servicios, ejecutan secuencias de comandos mediante un cliente de prueba y verifican las respuestas.

Ejecutar `scripts/cleanup.sh` (o `scripts/cleanup.ps1`) antes de cada tanda de pruebas garantiza un estado completamente limpio.

## Estructura prevista

```
tests/
├── test_suite.sh        # script principal de integración (Linux)
├── test_suite.bat       # script principal de integración (Windows)
├── happy_path/          # flujos normales de extremo a extremo
├── errors/              # entradas inválidas y condiciones de error
├── chains/              # ejecución de cadenas de programas
└── states/              # transiciones de estado de cada servicio
```

## Casos de prueba mínimos

### happy_path/
- Registrar un programa (`guardar`), leerlo y borrarlo.
- Crear un fichero (`crear`), actualizarlo con contenido, leerlo y borrarlo.
- Listar ficheros y programas con el almacén vacío y con elementos.
- Crear fichero de entrada y salida, ejecutar un lote de un solo programa, consultar estado hasta que termine y leer el fichero de salida.

### errors/
- Borrar un fichero con `refcount > 0` → `RESOURCE_BUSY`.
- Leer un ID inexistente (`f-9999`, `p-9999`, `l-9999`) → `NOT_FOUND`.
- Enviar JSON malformado al pipe → el servicio responde `INVALID_REQUEST` y sigue en pie.
- `ejecutar` con un programa que no existe en `aralmac` → `NOT_FOUND`.
- `guardar` un programa con ruta de ejecutable inválida → `INVALID_EXECUTABLE`.
- Acción desconocida en campo `action` → `INVALID_ACTION`.
- `matar` un lote ya terminado → `INVALID_REQUEST`.

### chains/
- Cadena de 1 programa: funciona igual que un lote simple.
- Cadena de 3 programas: verificar que la salida de cada programa es la entrada del siguiente.
- Cadena donde el segundo programa falla: el lote debe marcarse como `fallido` y los demás procesos deben terminarse.
- `ejecutar` con `id_fichero_entrada` inexistente → `NOT_FOUND`, no se lanza ningún proceso.

### states/
- `gesfich` suspendido → todas las operaciones (incluyendo `leer`) devuelven `SERVICE_SUSPENDED`.
- `gesprog` suspendido → `leer` funciona; `guardar`, `actualizar` y `borrar` devuelven `SERVICE_SUSPENDED`.
- `ejecutor` suspendido → `ejecutar` y `matar` devuelven `SERVICE_SUSPENDED`; `estado` funciona.
- `ejecutor parar` → no acepta nuevos lotes; espera que los activos terminen antes de salir.
- `ejecutor terminar` → mata todos los lotes activos inmediatamente y sale.
- `ctrllt terminar` → propaga `terminar` a todos los servicios y luego termina.
