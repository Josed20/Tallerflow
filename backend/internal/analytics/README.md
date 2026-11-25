# Métricas y telemetría S5

## Fuentes y significado

La consola distingue tres fuentes. El resumen operativo usa altas de talleres y
órdenes persistidas, y actividad explícita; no usa `user_sessions.last_seen_at`
como prueba de actividad. Los eventos de uso proceden del cliente. Los reportes
de instalaciones proceden exclusivamente de CSV Google Play. Una primera apertura
observada no es una descarga ni una persona única.

`GET /api/v1/console/analytics` admite `start`/`end` inclusivos YYYY-MM-DD,
`channel=web|android`, `version` e `include_demo=true`. Las fechas y límites son
UTC. El rango máximo es 366 días inclusivos dentro de los últimos 25 meses.
`totals.active_users` y `totals.active_workshops` cuentan DISTINCT sobre el rango
completo, nunca suman únicos diarios. Si el rango precede la cobertura o retención
de eventos, los totales son null. El día inicial de cobertura puede ser parcial;
`coverage_started_at` da el instante exacto. Los eventos no constituyen un censo:
opt-out, DNT/GPC, clientes antiguos, red y cierre de app pueden reducir cobertura.

El checkbox demo excluye talleres marcados explícitamente mediante una mutación
administrativa auditada. No se infiere demo por nombre/correo. Las primeras
aperturas anónimas y los agregados Play carecen de asociación con un taller y no
pueden excluir demos; la interfaz lo señala. Fotos cuenta registros persistidos,
incluidos `object_key` de R2; no verifica inventario físico del bucket. Cuando hay
objetos R2, `photo_bytes=null`: no conocemos sus bytes y no presentamos cero.
Los bytes de fotos antiguas representan el tamaño almacenado en la BD.

## Eventos e identificadores

Cada evento contiene exclusivamente UUID v4 aleatorio de evento e instalación,
tipo, fecha UTC de ocurrencia, canal y versión. El servidor añade recepción y, en
eventos autenticados, usuario/taller derivados de sesión. No acepta esos IDs de
tenant/usuario del cuerpo. No almacena correo, IP, ubicación, anuncios, rutas,
teclas pulsadas, archivos ni contenido de órdenes. El limitador usa temporalmente
IP en memoria para limitar solicitudes; no la guarda en el evento.

* `first_open`: una primera ejecución por almacenamiento local de app/navegador.
  Ruta anónima `/api/v1/analytics/first-open`, índice único por instalación.
* `registration`: verificación de correo del dueño completada, con evidencia
  `users.email_verified_at` cercana y una sola vez por taller. No significa
  formulario enviado ni ocurre cada login.
* `login`: requiere `USER_SIGNED_IN` del mismo actor/taller/canal en ±5 minutos;
  cada entrada de auditoría solo puede producir un evento. La selección excluye
  evidencia ya consumida y serializa las reclamaciones por usuario/taller para
  no perder un segundo login legítimo cercano. Usa fecha del servidor.
* `activity`: interacción real pointer/teclado con sesión, visible y fuera de
  consola/seguimiento. El cliente limita a una por minuto. No captura lectura
  pasiva, contenido de teclas ni detalle de la acción. Es una señal cliente,
  no una prueba de operación de negocio.

Los tres últimos usan `/api/v1/analytics/events` con auth y CSRF web o bearer
móvil; se verifica pertenencia activa y el tenant de la transacción. RLS permanece
forzado. El rol app no puede insertar directamente ni leer eventos de otro tenant;
las funciones SQL validan evidencia. La consola solo recibe agregados a través de
funciones SECURITY DEFINER con search_path fijo; PUBLIC no tiene EXECUTE.
Máximo 2048 bytes/evento, ventana de ocurrencia 7 días atrás / 5 minutos adelante,
120 eventos por IP/minuto por proceso y capacidad del limitador de 10 000 claves.
No se garantiza defensa distribuida contra instalaciones anónimas fabricadas.

Android lee `App.getInfo().build`: es App Version Code (3 para release 1.1.0),
no versionName. Web usa `VITE_APP_VERSION` explícito o la versión del paquete que
inyecta Vite (actualmente `1.1.0`). `unknown` solo queda como respaldo si se ejecuta
el módulo fuera de esa configuración de build. Una versión de nombre Play no
equivale a su código numérico.

## Almacenamiento local y privacidad

`localStorage` solo añade `tallerflow.analytics.v1` con instalación, marca de
apertura y cola; `tallerflow.analytics.disabled="true"` registra exclusión local.
La cola contiene evento y clave opaca usuario/taller para impedir atribuir un
evento pendiente a otra cuenta. No persiste password, bearer, sesión ni CSRF.
Máximo 100 eventos y siete días; reintentos conservan UUID. Los IDs de eventos
deduplican en el servidor. Respuestas permanentes inválidas se descartan; errores
de red, 429 y 5xx reintentan. El servidor también deduplica evidencia auth.

`getAnalyticsDisabled()` / `setAnalyticsDisabled(boolean)` se importan de
`frontend/src/native/analytics.ts` sin iniciar telemetría. El setter desactiva y
borra cola/identidad, notifica al runtime para vaciar memoria y abortar el fetch
pendiente. No puede retirar un evento que el servidor ya recibió. Al reactivar
se crea identidad nueva. También se respetan DNT=1 y GPC. El ID y la marca local
persisten mientras se conserve ese almacenamiento y la preferencia siga activa;
no tienen una promesa de caducidad de 90 días.

Los eventos crudos de servidor se purgan al superar 90 días desde ocurrencia;
Play se purga al superar 25 meses por fecha de reporte. Una tarea arranca con la
API y se ejecuta cada 24h, timeout 30s, errores registrados y reintento al día
siguiente. La limpieza requiere API en ejecución y las migraciones aplicadas;
no es una promesa de borrado instantáneo ni cubre copias de seguridad. La auditoría
administrativa conserva metadatos de importación, sin copiar las cifras del CSV.

## Importación Play y permisos pendientes

`POST /api/v1/console/analytics/play/import` solo existe detrás del host de consola,
sesión de administrador y CSRF. Acepta JSON `{csv, source_name}`; el archivo se
decodifica en navegador y el parser servidor soporta UTF-8 y UTF-16 LE/BE con BOM.
Límites 4 MiB / 10 000 filas, paquete `pe.tallerflow.app`, fechas no futuras dentro
de retención. Headers de reporte de instalaciones oficial en inglés, sin columnas
repetidas. Admite totales o `App Version Code`; rechaza país/dispositivo u otras
dimensiones para no duplicar instalaciones. Nunca obtiene datos por URL arbitraria.

Campos importados: `Daily Device Installs`, `Daily User Installs`, `Daily Device
Uninstalls`. Una columna omitida conserva su valor anterior al actualizar una fila
existente; en una fila nueva sigue siendo null. Una columna presente con celda
vacía, `-` o `N/A` reemplaza explícitamente su valor por null; cero explícito es 0.
Los códigos App Version Code se normalizan numéricamente antes de guardar y
detectar duplicados: `03` y `3` son la misma versión, no dos filas. Fechas ausentes
son null en dashboard. Cada fila guarda fuente fija `google_play_csv`, fecha de
reporte, código de versión/dimensión, nombre de reporte, instante de importación,
UUID de admin y SHA-256. Upsert atómico por fecha/paquete/dimensión/versión; importar
otra vez no suma cifras. La auditoría forma parte de la misma transacción. Si hay
totales y desglose, se usan totales para todas las versiones; al filtrar se usa
solo el desglose correspondiente. No se suman totales y versiones.

La importación automática no está habilitada: no se han proporcionado URI del
bucket de reportes ni credenciales autorizadas para consultarlo. No se inspeccionan
ni inventan datos de cuentas externas. Para habilitarla hará falta confirmar el
bucket `pubsite_prod_rev_*`, identidad de servicio y permisos Play de lectura de
información de app/reportes que correspondan; no hacen falta permisos de publicación
para esta función. No hay cambios de permisos Google/GitHub en esta implementación.

Formato y permisos se contrastaron con documentación oficial:
https://support.google.com/googleplay/android-developer/answer/6135870?hl=en

## Verificación local

Pruebas Go en `internal/analytics` y `internal/console`: validación, UTF-16,
nulos/cero, dimensión y paquete, duplicados, números, fechas, filtros acotados y
regresión de nombres GORM `created_last_7_days`/`created_last_30_days` y similares.
El arranque se importa dinámicamente desde `frontend/src/main.ts` después de
configurar Pinia y transporte; no se importa como efecto lateral del transporte
móvil. Una prueba importa el entrypoint real en web y Android y verifica ese orden.
Frontend: cola persistente/reintentos, no atribuir entre cuentas, expiración y
límites, preferencias, cancelación y salida SPA desde seguimiento. La prueba e2e
onboarding permite solo claves explícitas de telemetría y comprueba esquema y
ausencia de credenciales persistidas. Las pruebas globales y migración/runtime
PostgreSQL las coordina el parent; no duplicar compilaciones Go en paralelo.
