# Sprint 5: revisión de diseño y experiencia de uso

> Estado actualizado al 25/11/2026: la implementación sí fue subida a origin
> en feat/s5-stephano-operacion-metricas (e5d191d). Las menciones a trabajo
> exclusivamente local describen las validaciones anteriores. No hubo merge,
> PR, despliegue ni actualización de Play. Véase sprint-05-continuidad.md.

Fecha: 25 de noviembre de 2026.
Estado: revisión local, pendiente de validación del usuario.
Rama: `feat/s5-stephano-operacion-metricas`.

## Cambios

| Área | Resultado |
| --- | --- |
| Identidad visual | DM Sans alojada con el proyecto, cifras normales y componentes con colores y espaciado coherentes. |
| Portada | Explicación breve del sistema, capturas reales de escritorio y móvil, galería ampliable, fotografías referenciales y enlace a Google Play. |
| Cuenta | Privacidad, eliminación y términos en páginas internas con navegación de regreso. |
| Orden | Asignación de responsable independiente de la pausa de producción. Pausar exige un motivo; reanudar conserva la etapa y el responsable. |
| Seguimiento | Encabezado compacto, explicación de la etapa y actualización automática sin botón de actualización manual. |
| Tablero | Tarjetas con jerarquía de producto, cliente, entrega y cantidad; filtros y acciones accesibles. |
| Panel del taller | Indicadores de pedidos activos, atrasados, en riesgo y completados; carga por etapa y filtros de trabajo. |
| Consola | Resumen de plataforma, gráfico por métrica, estados de talleres en español y ventanas de gestión con confirmación. |
| Demostraciones | Control de clasificación visible y conservación de los filtros de métricas al modificarlo. |
| Avisos | Confirmaciones flotantes que se cierran automáticamente; errores persistentes con cierre manual. |

## Criterios de datos

Las capturas de la portada corresponden a Confecciones Demo, un taller ficticio
creado exclusivamente en la instalación local. Las fotografías de costura son
referenciales y tienen sus fuentes identificadas en
`frontend/public/product/README.md`. No se presentan como producción de un cliente.

El gráfico conserva los días sin cobertura como datos faltantes y no une la
serie a través de esos días. Los máximos se calculan con las observaciones
recibidas, incluidos los ceros reales. Las instalaciones de Google Play se
importan desde los reportes CSV oficiales; las aperturas no se presentan como
descargas. La descarga automática de esos reportes sigue requiriendo acceso
autorizado al origen de Google Play.

## Comprobaciones funcionales

- Se recorrieron la portada, la galería, las páginas internas de cuenta y los
  enlaces de regreso.
- La asignación de un responsable se guardó sin pausar la orden. Se comprobó
  después la pausa con motivo y la reanudación sin perder el responsable.
- Se verificaron el indicador demo, la suspensión y reactivación, y la programación
  y cancelación de eliminación sobre el taller local de demostración. No se
  ejecutó eliminación definitiva; el taller terminó activo y sus datos permanecen.
- La navegación de teclado permite cambiar de captura y cerrar las ventanas con
  Escape. Las ventanas emplean el comportamiento modal nativo del navegador.
- La revisión de anchos incluyó 1440, 390 y 360 px. Se corrigieron los desbordes
  del encabezado del panel, la guía inicial y una etiqueta accesible de la tabla
  de administración que escapaba del contenedor horizontal.
- La consola solicita iniciar sesión cuando una pestaña no dispone del token de
  seguridad necesario para las acciones administrativas.

## Evidencias locales

Los archivos `tmp/local-review/new-*.png` y los registros
`tmp/local-review/redesign-*.log` contienen las comprobaciones de esta revisión.
Las capturas del producto destinadas a la portada se conservan en
`frontend/public/product` junto con sus créditos.

Los accesos y la cuenta de prueba se detallan en `tmp/local-review/LEEME.txt`.
No se realizó push, PR, merge, despliegue ni actualización de Google Play.

## Resultado de la revisión final

- Suite de interfaz: 44 archivos y 196 pruebas aprobadas.
- Compilación: `vue-tsc -b` y `vite build` aprobados.
- Recorrido completo del Sprint 5 a 360 px: aprobado.
- Navegador real: portada, galería, enlaces de cuenta, panel, tablero y seguimiento
  aprobados a 1440, 390 y 360 px sin desbordes ni errores de página.
- Consola: clasificación demo, suspensión/reactivación y programación/cancelación
  de eliminación aprobadas con API local; vistas a 1440 y 390 px verificadas.
- Se comprobó la ampliación de capturas, el cierre de ventanas y la selección de
  métricas. No se ejecutó eliminación definitiva de talleres.
- `git diff --check` aprobado. No se publicaron los cambios.
