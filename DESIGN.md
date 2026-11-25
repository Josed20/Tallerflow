# TallerFlow: dirección visual

Actualización: 25 de noviembre de 2026.

La interfaz prioriza el trabajo de un taller de confección: pedidos, etapas,
entregas, evidencias y comunicación con el cliente. Los cambios de esta revisión
permanecen en local hasta la validación del usuario.

## Identidad

- Azul marino `#092B55`: navegación, identidad y portada.
- Azul `#103C73`: acciones principales y selección.
- Turquesa oscuro `#168271`: avance, actividad y acentos legibles.
- Blanco `#FFFFFF` y gris `#F4F6FB`: superficies y fondo de trabajo.
- Verde, ámbar y rojo: estados de entrega acompañados de texto.
- Tipografía DM Sans, alojada con el proyecto bajo licencia SIL OFL.
  Se usan cifras normales; los códigos de orden no requieren una fuente monoespaciada.

## Jerarquía y composición

Los títulos utilizan pesos 650–750 y espaciado moderado. Se evita la expansión
artificial de letras y las mayúsculas en párrafos. Las superficies usan bordes
suaves y radios de 12–18 px; el color expresa estado o selección.

La portada muestra pantallas reales del producto y fotografías de referencia.
Los pedidos y el taller de las capturas se identifican como datos de demostración.
No se publican cifras de adopción, testimonios ni calificaciones inventados.

## Flujos

- Mi cuenta ofrece accesos a privacidad y eliminación en páginas internas con
  navegación de regreso. El contenido no se despliega debajo de los botones.
- Responsable del pedido y estado de producción son funciones independientes.
  Pausar exige un motivo; reanudar conserva la etapa y el responsable.
- El seguimiento se consulta automáticamente mientras la página está visible.
  Solo se ofrece reintentar cuando falla la carga inicial.
- El tablero muestra producto, cliente, fecha, cantidad y estado. Completar una
  etapa conserva la confirmación del flujo existente.
- La consola ofrece una ventana de administración por taller. Los cambios de
  acceso requieren motivo y confirmación. La clasificación demo usa un control
  de estado visible y solo cambia tras la respuesta del servidor.

## Datos y gráficos

El panel del taller presenta pedidos activos, atrasados, en riesgo y completados,
además de la carga por etapa. Las cantidades de distintas unidades no se suman.

La consola representa actividad por día y permite elegir la métrica. Los días
sin cobertura permanecen sin datos: no se transforman en ceros ni se unen en el
gráfico. Las instalaciones oficiales de Play siguen separadas de las aperturas.

## Accesibilidad y adaptación

Se conservan etiquetas, estados de carga, foco visible, navegación con teclado,
texto junto al color y ventanas con foco contenido y cierre mediante Escape.
Los controles principales tienen áreas de interacción de al menos 44 px. Las
tablas y el tablero desplazan su contenido dentro del contenedor, sin expandir
el ancho de la página. El movimiento respeta la preferencia de reducción.

## Referencias consultadas

- [TuJuegoYa](https://tujuegoya.com/): presentación visual del producto y contexto de uso.
- [Linear: Board layout](https://linear.app/docs/board-layout): jerarquía del tablero.
- [UI/UX Pro Max](https://github.com/nextlevelbuilder/ui-ux-pro-max-skill): interacción, accesibilidad y gráficos.
- Los créditos de fotografías y capturas se conservan en
  `frontend/public/product/README.md`.
