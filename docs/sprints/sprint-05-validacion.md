# Sprint 5: validación local

> Estado actualizado al 25/11/2026: la implementación sí fue subida a origin
> en feat/s5-stephano-operacion-metricas (e5d191d). Las menciones a trabajo
> exclusivamente local describen las validaciones anteriores. No hubo merge,
> PR, despliegue ni actualización de Play. Véase sprint-05-continuidad.md.

Fecha: 24 de noviembre de 2026 (Lima).
Rama: `feat/s5-stephano-operacion-metricas`.
Base remota comprobada: `1aa7f4ee2bb2319c36bc88bcfe4f18a29e796ffb`.

## Estado de entrega

Los cambios se conservan exclusivamente en local para la revisión del usuario.
No se realizó push, PR, merge, despliegue ni actualización de Google Play.
Los permisos del entorno `production` mantienen la aprobación de José.

## Comprobaciones aprobadas

| Comprobación | Resultado observado |
| --- | --- |
| Frontend | 42 archivos y 191 pruebas aprobadas, repetidas con Node 24 y Vite 6.4.6. |
| Compilación web | `vue-tsc -b` y `vite build` aprobados. |
| Backend | `go test -p 1 ./...`, `go vet ./...` y suite completa `go test -race -p 1 ./...` aprobados. |
| PostgreSQL | Integración de analytics, autenticación, consola, órdenes, recuperación y registro aprobada. |
| Migraciones | Instalación desde cero hasta V21 y actualización a V22 aprobadas. |
| Contrato de producción | Backfill histórico bajo FORCE RLS, aislamiento, 20 posiciones, snapshots y purga aprobados. |
| Contrato de primer propietario | Purga real del usuario y conservación del marcador de instalación reclamada aprobados. |
| Navegador | 22 recorridos completos aprobados con Vite 6.4.6: seguimiento, operaciones, recuperación, registro con Mailpit y Sprint 5. |
| Alta inicial | Validación previa aprobada con servicios locales; ejecución observada de 7,5 segundos. |
| Consola real | Login local y consultas de resumen, talleres y métricas con HTTP 200. Capturas a 1440 px y 390 px sin desborde global. |
| Android | Compilaciones debug, release con R8 y AAB aprobadas. Manifiesto del escáner sin orientación fija. |
| Celular | `pe.tallerflow.app.debug`, versión 1.1.0-local, instalada y abierta por USB. Se verificó el inicio de sesión por captura. |
| Docker | Imágenes backend y frontend construidas; API nonroot con `ready` y frontend Nginx con HTTP 200. |
| OpenAPI | Cuatro contratos válidos; seis recomendaciones no bloqueantes sobre metadatos y respuestas. |
| Dependencias | Vite actualizado de 6.4.3 a 6.4.6. Auditoría: 0 altos/críticos y 0 avisos en dependencias de producción. |
| Git | `git diff --check` aprobado; documentos previos, diagramas y videos del usuario excluidos del cambio. |

Las pruebas de navegador usaron perfiles aislados. Las pruebas reales solo
modificaron PostgreSQL y Mailpit locales, con datos ficticios. Las cifras de
validación no representan talleres ni descargas de producción.

## Correcciones verificadas en esta sesión

- La cola de métricas guarda cada evento en una clave UUID independiente. Tres
  regresiones verifican escrituras entre pestañas y entregas en curso; el borrado
  de preferencias también elimina esos eventos. Los contratos E2E conservan la
  comprobación de que no se almacenan credenciales.
- El resumen de consola separa cifras y etiquetas en tarjetas. La tabla de
  actividad tiene desplazamiento contenido y las vistas móviles mantienen las
  tarjetas dentro de su ancho disponible.
- La aplicación de pruebas se instala junto a la de Play Store y usa los servicios
  locales por USB. Los servidores web de revisión solo escuchan en loopback.
- Las carpetas generadas de Android se excluyen del observador de Vite para evitar
  recargas mientras se compila. Las compilaciones se ejecutan por turnos; Go tiene
  un límite de memoria de 512 MiB durante la construcción de la imagen.

## Pendientes antes de publicar

1. Revisión del usuario y autorización posterior para enviar los cambios.
2. Prueba física de cámara, escáner y rotación con el dispositivo; el arranque ya
   fue comprobado, pero no se presenta esa comprobación como prueba de hardware.
3. Firma con la clave existente y envío del AAB después de la aprobación. El AAB
   local de validación no se presenta como un paquete firmado de producción.
4. Actualización de Seguridad de los datos en Play para las métricas incluidas.
   Los avisos reales de Google se revisaron: R8 y orientación tienen cambios;
   la biblioteca de cámara aún contiene llamadas obsoletas, como detalla el
   runbook Android. Su desaparición debe comprobarse tras el análisis en Play.
5. La descarga automática de informes oficiales Play requiere acceso al bucket
   y credenciales autorizadas. La importación manual CSV está implementada.
6. Permanecen ocho avisos moderados en herramientas de desarrollo; su resolución
   completa requiere una actualización mayor y su validación, que no se aplicó
   automáticamente durante esta entrega.

Los accesos y pasos de revisión están en `tmp/local-review/LEEME.txt`, fuera de
Git. No se copiaron cuentas, claves ni datos de producción al entorno local.

## Ajustes posteriores a la revisión visual — 25 de noviembre de 2026

Esta revisión también permanece exclusivamente en local. No reemplaza la
validación física de Android ni la revisión del usuario antes de publicar.

- La barra lateral tiene un único acceso a Mi cuenta mediante el perfil. La
  política de privacidad y las instrucciones de eliminación se muestran dentro
  de esa pantalla, con la navegación del taller disponible.
- La configuración de etapas se encuentra en Configuración del taller. El panel
  principal mantiene la información de producción, entregas y pedidos en pausa.
- Los contactos de clientes se presentan como texto. Se retiró la columna Estado.
  Archivar y restaurar conserva las órdenes; no se agregó borrado definitivo.
- Cada orden nueva crea su enlace de seguimiento en la misma transacción. La
  interfaz ofrece únicamente copiarlo; recuperarlo no cambia el enlace existente.
  La prueba PostgreSQL verifica también idempotencia y reversión de la creación
  cuando falla la asociación con el actor.
- La coordinación del pedido es opcional y se presenta plegada. El responsable
  es un miembro del equipo y la pausa documenta un impedimento para continuar.
- Los avisos de acciones y errores reutilizables se muestran flotantes y se pueden
  cerrar. Los estados persistentes, instrucciones y errores de campos conservan
  su contexto junto al contenido correspondiente.
- El seguimiento usa títulos legibles, datos del taller sin recuadros decorativos,
  descripción de la etapa actual y recorrido cronológico. La consulta automática
  no genera avisos; la actualización manual sí informa el resultado.
- La consola mantiene filtros, métricas y auditoría con jerarquía tipográfica y
  tablas cuyo desplazamiento se limita a su contenedor.

Comprobaciones de esta revisión:

| Alcance | Resultado |
| --- | --- |
| Interfaz | 42 archivos, 192 pruebas aprobadas con Node 24. |
| Servicios | Pruebas Go de órdenes y API aprobadas. |
| PostgreSQL | Ciclo de enlace estable y creación automática aprobados contra la base local. |
| API real | Enlace presente antes de copiar; dos recuperaciones iguales; archivo/restauración conserva la orden. |
| Recorrido Sprint 5 | Aprobado a 360 px con configuración, snapshots y seguimiento público. |
| Navegador local | Clientes, cuenta integrada, configuración, copia del enlace y panel revisados a 1440, 390 y 360 px. |

Las evidencias de esta revisión están en `tmp/local-review/design-*.png` y los
registros en `tmp/design-*.log`; son archivos locales ignorados por Git. Los datos
utilizados son ficticios. La plantilla local se dejó con siete etapas habituales
para facilitar la revisión; las órdenes anteriores conservan su propio proceso.
La compilación final (`vue-tsc` y `vite build`) fue aprobada. Clientes, cuenta y métricas se verificaron nuevamente tras los últimos ajustes.

## Rediseño de interfaz — validación final del 25/11/2026

La revisión descrita en `sprint-05-diseno.md` reemplaza la experiencia anterior de
privacidad desplegable y coordinación agrupada. Cuenta con 196 pruebas de
interfaz aprobadas, compilación web aprobada y recorrido Sprint 5 aprobado. Las
capturas de revisión y las pruebas de administración se realizaron contra la
instalación local; no se enviaron cambios a GitHub ni a producción.
