# Continuidad del Sprint 5 — TallerFlow

Actualizado: 25 de noviembre de 2026, America/Lima.

## Punto de partida para otra IA

El trabajo está en `feat/s5-stephano-operacion-metricas`, subido a `origin`.
Los commits de implementación son:

- `586eb54`: operación, seguimiento y métricas de plataforma.
- `e5d191d`: rediseño de experiencia, correcciones y validación de publicación.

La base es `1aa7f4ee2bb2319c36bc88bcfe4f18a29e796ffb`. No se hizo merge, PR,
despliegue ni actualización de Google Play. El usuario debe revisar y aprobar los
cambios antes de integrarlos o publicarlos. La subida de ramas no equivale a esa aprobación.
Este documento se crea después de esos commits y queda local hasta una nueva subida.

No se debe reiniciar el desarrollo desde main ni sobrescribir cambios locales.
Primero revisar `git status`, la rama y los documentos enlazados al final.

## Funcionalidad implementada

| Área | Resultado |
| --- | --- |
| Producción | Configuración de 1 a 20 etapas con descripción para el cliente. Cada orden conserva una copia de su proceso; cambiar la plantilla no modifica órdenes anteriores. |
| Órdenes | Responsable opcional del equipo; asignación separada de pausar/reanudar. La pausa requiere motivo y bloquea completar etapas. Historial de cambios con actor, fecha y motivo; control de edición concurrente. |
| Seguimiento | Enlace creado automáticamente en la misma transacción de la orden. Copiar recupera el mismo enlace; no hay botones de WhatsApp ni de ver como cliente. Vencimiento 30 días después de completar la orden. |
| Vista pública | Encabezado compacto, explicación de etapa, recorrido y fotos. Actualización automática sin botón manual. No revela responsable ni motivo interno de pausa. |
| Clientes | Teléfono y correo como texto, buscador corregido y sin columna Estado. Archivo/restauración conserva las órdenes; no se implementó borrado definitivo. |
| Exportación | CSV de órdenes y clientes con autorización y protección frente a fórmulas. No confundir con un exportador XLSX. |
| Panel del taller | Indicadores de activos, atrasados, en riesgo y completados; carga por etapa, entregas y filtros. Las cantidades de unidades distintas no se mezclan. |
| Cuenta | Un único acceso desde el perfil. Privacidad, eliminación y términos en rutas internas que conservan la navegación. Se mantiene la página pública de eliminación exigida para la ficha de Play. |
| Consola | Resumen, métricas por período, gráfico seleccionable, clasificación demo y gestión de talleres mediante ventanas con confirmación. Suspender/reactivar y programar/cancelar eliminación. Se conserva la protección de purga definitiva. |
| Telemetría | Eventos de uso con elección del usuario, deduplicación y cola por evento UUID. Diferencia aperturas, actividad y descargas oficiales. |
| Google Play | Importación manual de CSV oficiales de instalaciones. Los días sin cobertura se distinguen de ceros reales. La descarga automática de informes no está operativa sin acceso autorizado al origen. |
| Diseño | DM Sans local con licencia; componentes, tarjetas, tablero, panel y consola renovados. Avisos flotantes; confirmaciones se ocultan automáticamente y errores tienen cierre manual. |
| Landing | Explicación del producto, galería de capturas de escritorio/móvil y enlace a Play. Capturas de taller ficticio y fotografías referenciales con créditos. |
| Android | Versión prevista 1.1.0, código 3; R8 y reducción de recursos; Material actualizado y escáner sin orientación fija. Variante debug separada de producción. |
| Plataforma | Migraciones hasta V22, contratos OpenAPI, ajustes de Docker y CI; Node 24 y Vite 6.4.6 para las verificaciones. |

## Validación realizada

Estas son comprobaciones anteriores documentadas, no una nueva ejecución al redactar este archivo.

- Frontend final: 44 archivos y 196 pruebas aprobadas.
- `vue-tsc -b` y compilación Vite aprobados.
- Recorrido Sprint 5 a 360 px aprobado.
- Portada, galería, cuenta, panel, tablero y seguimiento revisados a 1440, 390 y 360 px sin desbordes ni errores de página.
- Consola: clasificación demo, suspensión/reactivación y programación/cancelación de eliminación probadas con API local. No se ejecutó purga definitiva en esa revisión visual.
- Pruebas Go, integración PostgreSQL, migraciones y contratos operativos documentados en el informe de validación.
- Creación atómica del enlace: idempotencia y reversión de transacción comprobadas.
- Android debug/release/AAB compilados. Se instaló la variante debug y se comprobó su apertura; eso no acredita la prueba física completa de cámara y escáner.
- Auditoría registrada: sin avisos altos/críticos ni avisos de producción; ocho moderados de herramientas de desarrollo pendientes.

Evidencias locales: `tmp/local-review/redesign-*.log` y capturas en esa carpeta.
No están en Git. Las evidencias de pruebas locales no son métricas de clientes reales.

## Pendientes, en orden recomendado

1. **Revisión del usuario:** validar diseño y flujos locales. Corregir únicamente lo que se detecte; no presentar la implementación como aceptada todavía.
2. **Prueba física Android:** cámara, permisos, selección/subida de fotos, escáner, rotación y regreso a la app; comprobar pantalla grande cuando haya dispositivo disponible. Registrar resultados reales.
3. **Avisos de Play:** contrastar los avisos actuales con el AAB nuevo. R8 y orientación tienen correcciones, pero requieren análisis de Google. La biblioteca de cámara conserva llamadas obsoletas; no afirmar que todos los avisos desaparecieron.
4. **Seguridad de los datos:** revisar la declaración con la telemetría implementada, su finalidad, identificadores y elección del usuario antes de distribuir la actualización.
5. **Informes automáticos Play:** configurar solo con bucket e identidad autorizados. Mientras tanto usar importación manual CSV. El código propio no permite contar descargas exactas que Google aún no ha reportado.
6. **Dependencias de desarrollo:** evaluar los ocho avisos moderados y el impacto de una actualización mayor. Repetir auditoría antes de decidir; el dato registrado no sustituye una consulta actual.
7. **Integración y publicación, únicamente tras autorización:** revisar diferencias con main actualizado sin perder trabajo, ejecutar comprobaciones pertinentes, integrar, desplegar con el entorno protegido y aprobación existente de José. El usuario decidió no cambiar aprobadores.
8. **Android de producción:** usar firma existente, comprobar que el código de versión supera el publicado, generar/subir AAB y verificar revisión de Google. Desplegar la web no actualiza el código empaquetado en Android.
9. **Después de publicar:** verificar salud, migraciones, flujos web y móvil, consentimiento y cobertura de métricas; actualizar documentación con los resultados observados.

## Entorno local: referencias, no garantía de servicios activos

En la última revisión se usaron estos destinos:

- Aplicación: `http://127.0.0.1:5180`.
- Consola: `http://127.0.0.1:5181/consola`.
- API: `http://127.0.0.1:18084`, disponibilidad en `/health/ready`.
- PostgreSQL: puerto 55435. Mailpit: interfaz 18025, SMTP 11025.

Comprobar procesos y contenedores antes de arrancar nuevos servicios; no asumir
que sobrevivieron al cierre de la sesión. La configuración local de consola está
en `tmp/local-review/vite-console.config.mjs`. Los accesos de prueba están en
`tmp/local-review/LEEME.txt`; no copiarlos a documentación ni Git.

Desde `frontend`, con Node 24: `npm test` y `npm run build`.
Las pruebas E2E requieren revisar primero su configuración y servicios locales.
Compilar de forma secuencial para evitar saturar la memoria del equipo.

## Restricciones y archivos que deben conservarse

- No hacer merge, PR, despliegue ni actualización de Play sin autorización nueva.
- Usar el patrón existente `feat/s5-...` o `fix/...`; no crear ramas `codex/`.
- `diagrams/`, `docs/validacion/` y `video/` son archivos previos del usuario no incluidos en estos commits. No borrarlos ni añadirlos indiscriminadamente.
- `tmp/` contiene configuración y accesos locales: mantenerlo ignorado. Nunca publicar secretos, contraseñas, TOTP o archivos de firma.
- Las ramas antiguas `feat/s4-stephano-apk-release` y `feat/s4-stephano-r2-photos` se subieron por error de alcance. El usuario fue informado; no contienen esta entrega y no se han eliminado. No eliminarlas sin instrucción expresa.
- No confundir pruebas ficticias, primeras aperturas o talleres demo con adopción real.

## Documentación de referencia

- [Operación y métricas](sprint-05-operacion-metricas.md): reglas funcionales, fuentes y contratos.
- [Diseño](sprint-05-diseno.md): experiencia final que reemplaza las versiones visuales anteriores.
- [Validación](sprint-05-validacion.md): pruebas, evidencias y límites.
- [Android](../runbooks/mobile-android.md): compilación y advertencias de Google.
- [Analytics técnico](../../backend/internal/analytics/README.md): eventos e importación de informes.

## Instrucción sugerida para continuar

Leer este documento y sus referencias. Verificar rama, cambios y servicios locales.
Retomar los pendientes desde la validación del usuario y Android físico; informar
qué puede comprobarse y qué requiere accesos o dispositivo. No fusionar, desplegar
ni publicar una versión por asumir que la subida previa de la rama lo autoriza.
