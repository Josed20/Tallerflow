---
# gstack: design-md-format=spec
name: TallerFlow
description: Etiqueta tejida. La interfaz es la prenda, clara y plana; la marca es la etiqueta marina cosida con hilo turquesa; el único color que grita es el de un pedido atrasado.
colors:
  primary: "#0A346F"
  primary-hover: "#154A8F"
  primary-strong: "#072756"
  primary-soft: "#E7EEF8"
  primary-border: "#A9BCD9"
  on-primary: "#FFFFFF"
  shell: "#062550"
  shell-raised: "#0D3F85"
  accent: "#00BFB2"
  on-accent: "#04121F"
  accent-text: "#006B64"
  surface: "#FFFFFF"
  background: "#F3F5F9"
  text: "#16233D"
  text-muted: "#556074"
  border: "#D5DCE6"
  focus: "#0A346F"
  success: "#0B6B3A"
  success-soft: "#E2F3E9"
  warning: "#FFB020"
  on-warning: "#3D2600"
  error: "#C8102E"
  on-error: "#FFFFFF"
  error-soft: "#FCE6E9"
  completed: "#5A5C72"
  completed-soft: "#ECEDF2"
typography:
  display:
    fontFamily: Archivo
    fontWeight: 900
    fontSize: clamp(1.55rem, 7.5vw, 3.4rem)
    letterSpacing: -0.005em
    fontVariation: '"wdth" 125'
  headline:
    fontFamily: Archivo
    fontWeight: 800
    fontSize: 1.625rem
    fontVariation: '"wdth" 125'
  body:
    fontFamily: Atkinson Hyperlegible Next
    fontSize: 1rem
    lineHeight: 1.55
  label:
    fontFamily: Archivo
    fontWeight: 700
    fontSize: 0.72rem
    letterSpacing: 0.07em
    textTransform: uppercase
  mono:
    fontFamily: Atkinson Hyperlegible Mono
    fontFeature: tnum
rounded:
  sm: 4px
  md: 6px
  full: 9999px
spacing:
  xs: 4px
  sm: 8px
  md: 16px
  lg: 24px
  xl: 32px
  2xl: 48px
components:
  button-primary:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    rounded: "{rounded.md}"
    minHeight: 48px
  button-primary-hover:
    backgroundColor: "{colors.primary-hover}"
  button-secondary:
    backgroundColor: "{colors.surface}"
    textColor: "{colors.primary}"
    borderColor: "{colors.primary}"
    rounded: "{rounded.md}"
  button-danger:
    backgroundColor: "{colors.error}"
    textColor: "{colors.on-error}"
    rounded: "{rounded.md}"
  input:
    borderColor: "{colors.border}"
    rounded: "{rounded.md}"
    minHeight: 48px
  ficha:
    backgroundColor: "{colors.surface}"
    borderColor: "{colors.border}"
    rounded: "{rounded.sm}"
    notch: 14px
  status-on-time:
    backgroundColor: "{colors.success-soft}"
    textColor: "{colors.success}"
  status-at-risk:
    backgroundColor: "{colors.warning}"
    textColor: "{colors.on-warning}"
  status-late:
    backgroundColor: "{colors.error}"
    textColor: "{colors.on-error}"
  status-completed:
    backgroundColor: "{colors.completed-soft}"
    textColor: "{colors.completed}"
  nav-active:
    backgroundColor: "{colors.shell-raised}"
    textColor: "{colors.on-primary}"
    indicatorColor: "{colors.accent}"
  workshop-label:
    backgroundColor: "{colors.primary}"
    textColor: "{colors.on-primary}"
    borderColor: "{colors.accent}"
    rounded: "{rounded.sm}"
  link:
    textColor: "{colors.primary}"
    underlineColor: "{colors.accent}"
---

# TallerFlow

## Overview

**Creative North Star:** "Etiqueta tejida": industrial y utilitario con alma textil. La interfaz es la prenda (blanca, plana, legible bajo fluorescente) y la marca es la etiqueta tejida que el taller cose en cada prenda: una banda marina con hilo turquesa. La app toma sus formas de los objetos reales del taller (la ficha que viaja con cada lote, la costura, la etiqueta) y se lee con la claridad de un letrero desde el otro lado del pasillo.
**Lo que todo usuario debe recordar:** "Abro el celular y en 2 segundos sé qué pedido está atrasado." Cada decisión de este documento sirve a esa frase.
**Product context:** trazabilidad de órdenes para talleres de confección de Gamarra (Lima). Usuarios: dueño o administrador de taller de 10–30 operarios (cliente objetivo), dueño de taller tradicional con poca experiencia digital, operarios en planta con celulares Android de 360 px y la marca cliente que sigue su pedido por un enlace público.
**Mode per surface:**
- Panel, tablero, detalle de orden, equipo, clientes: Operate.
- Página pública de seguimiento (`/seguimiento/:token`): Read, con el estado primero.
- Landing de presentación: Persuade (debe alinearse a este sistema).
**Reference sites (investigación del 2026-10-17):** https://katanamrp.com, https://www.aptean.com/en-US/solutions/mes/products/apparel-shop-floor-control-exenta, https://gomalomo.com/order-tracking-pages-that-inspire, https://www.yape.com.pe. Ningún producto del rubro textil tiene voz visual propia: el software de planta es corporativo y gris, o un SaaS genérico.
**Vista previa aprobada:** la del 2026-10-17 (https://claude.ai/artifact/4wc2hv9KE4hf1WPQunj3i8) muestra la marca anterior; la del 2026-10-20 ("Etiqueta tejida") se aprobó en la sesión de cambio de marca.
**Key characteristics:**
- La marca calla (azul marino y grises, con el turquesa como hilo de firma) para que el rojo de "Atrasada" sea lo único que grita.
- El nombre de cada taller va en su propia etiqueta tejida: el multi-taller se ve, y la marca cliente sabe quién le cose.
- El panel abre con un titular, no con una grilla de métricas.
- Las órdenes tienen forma de ficha viajera, con una costura de 7 puntadas que muestra el avance.
- Cada estado se reconoce por palabra, forma y color.

## Colors

**Strategy:** Restrained. Un color de marca (el marino del logo) más neutros derivados de él, un hilo turquesa que firma y nunca informa, y el color fuerte reservado para el estado de las órdenes.
**Light or dark:** claro por defecto. Se usa bajo luz fluorescente de taller y de día, en celulares baratos. El tema oscuro existe para el dueño que revisa de noche.

- **Azul marino (`primary`, el del logo)** es la marca: botones principales, foco, progreso y la costura completada. Blanco sobre él da 12.1:1; el hover aclara a `primary-hover` (8.7:1). La barra lateral usa `shell`, un marino más profundo (15.1:1 con blanco), y el ítem activo `shell-raised`.
- **Hilo turquesa (`accent`)** es la firma de la marca, no un color de interfaz. Como texto sobre blanco da 2.3:1, así que nunca es texto: para texto de marketing o enlaces se usa `accent-text` (6.4:1). Sobre marino da 5.3:1 y sobre la barra 6.6:1; como relleno lleva `on-accent` encima (8.2:1).
  - Permitido: el logo; el orillo de 3 px en el borde de la barra lateral; el indicador del ítem activo; la etiqueta del taller; el subrayado de enlaces marinos; detalles de la landing.
  - Prohibido: chips de estado, la costura, la cifra de atrasadas, rellenos de botón, anillos de foco, fondos tintados en pantallas de operación, brillos y degradados. Con el verde de "a tiempo" solo da 2.9:1: si se acerca a un estado, se lee como "a tiempo".
- **Semáforo**, el corazón del sistema:
  - A tiempo: texto `success` sobre `success-soft`, con círculo.
  - En riesgo (vence en 48 h): relleno `warning` con texto `on-warning`, con triángulo.
  - Atrasada: relleno sólido `error` con texto blanco, con cuadrado y "!". Es el único relleno saturado de la interfaz; si algo más usa rojo sólido, el rojo deja de avisar.
  - Completada: grafito `completed` sobre `completed-soft`, con check. Pierde color a propósito: lo terminado se guarda y no compite con lo urgente.
- **Accesibilidad:** el rojo y el verde tienen casi la misma luminancia (1.12:1), así que una persona daltónica no los distingue por color. Por eso ningún estado depende solo del color. Todos los pares de texto superan 4.5:1 y el foco tiene 10:1 sobre el fondo.
- **Tema oscuro** (redefine tokens, no invierte): fondo `#0F1624`, superficie `#172236`, texto `#E7EDF6`, secundario `#A1AEC2`, borde `#2A3850`, marca `#7FA8E8` con texto `#0B1526` encima (7.5:1), barra lateral `#06182F`, a tiempo `#5DD39A` sobre `#123222`, atrasada mantiene `#C8102E` con texto blanco, completada `#9A9DB4` sobre `#222C3E`. El ámbar de riesgo no cambia. En oscuro el turquesa queda solo en la barra (logo, orillo, etiqueta): junto al verde oscuro de "a tiempo" da 1.1:1 y serían el mismo color.
- **Reemplaza** al azul mezclilla `#1F3A68` (Sprint 3), que a su vez reemplazó al verde petróleo `#00685f` y al foco `#6be7d6` (1.5:1, no cumplía). Los dos rojos (`--tf-error` y `--tf-danger`) siguen unificados en `error`.

## Typography

Tres familias gratuitas de Google Fonts, verificadas contra la API de Google Fonts el 2026-10-17:
`https://fonts.googleapis.com/css2?family=Archivo:wdth,wght@62..125,100..900&family=Atkinson+Hyperlegible+Mono:wght@200..800&family=Atkinson+Hyperlegible+Next:wght@200..800&display=swap`

- **Archivo, ancho expandido (`wdth` 125), 800–900:** titulares, el bloque de atrasadas y cifras grandes. Ancha y firme, como la rotulación de un fardo. Usar `font-variation-settings: "wdth" 125`, porque `font-stretch` no siempre responde con la carga de Google Fonts.
- **Archivo, ancho normal, 700, mayúsculas con 0.07em de espaciado:** etiquetas cortas ("Carga por estación"). El espaciado en mayúsculas pequeñas no se quita.
- **Atkinson Hyperlegible Next:** todo el texto y los botones. Diseñada para baja visión: distingue 0/O y 1/l/I, clave para operarios apurados y dueños con poca confianza digital.
- **Atkinson Hyperlegible Mono:** códigos `OT-000003`, fechas, horas y cantidades, con números del mismo ancho.
- **Wordmark:** "Taller" en marino y "Flow" en turquesa, en Plus Jakarta Sans 800 convertida en trazos. Vive solo dentro del logo SVG: no se carga como fuente ni se reescribe en Archivo, porque dos grotescas gruesas juntas pelean. Ningún titular se parte en marino y turquesa imitándolo. Entre el logo y un titular quedan al menos 32 px.
- **Escala:** cifra de alarma 1.55–1.8rem en celular (hasta 3.4rem en escritorio), título de pantalla 1.625rem, sección 1.125rem, texto 1rem, secundario 0.8rem, etiqueta 0.72rem. Los niveles se diferencian por tamaño y ancho, no solo por peso.

## Layout

- **Celular primero (360 px):** en el celular, la lista de órdenes es la vista por defecto; en escritorio (≥ 900 px), el tablero Kanban de 7 columnas más Completadas.
- **Panel del dueño, en este orden:** titular de atrasadas → las fichas urgentes → "Salen hoy" → carga por estación.
- **Página del cliente:** estado en grande ("Costura"), fecha prometida y semáforo, costura de avance, recorrido de estaciones y fotos.
- **Densidad:** cómoda para el operario (botones de 48 px como mínimo, acción principal fija abajo al alcance del pulgar) y compacta para el dueño en escritorio.
- **Espaciado:** base de 4 px; separaciones de 16 px entre bloques en el celular y de 24–32 px en escritorio. Margen lateral mínimo de 16 px.

## Elevation & Depth

Casi plano. La separación viene del borde (`border`) y del fondo (`background` frente a `surface`), no de sombras. Las únicas sombras permitidas son las de los diálogos y avisos flotantes, siempre con desplazamiento y desenfoque suave, nunca un brillo sin desplazamiento.

## Shapes

- **Radios:** 4 px en fichas, chips y tarjetas; 6 px en botones y campos; círculo completo solo en puntos de estado y del recorrido.
- **La ficha** lleva la esquina superior derecha recortada (14 px), hecha con `clip-path`. Es la forma propia del sistema: no usarla en elementos que no sean una orden.
- Nada de píldoras ni de radios grandes en todo.

## Components

- **Ficha de orden:** código en mono, chip de estado arriba a la derecha (antes del recorte), producto, estación actual en negrita, cliente y fecha. Abajo, la costura de 7 puntadas. Si está atrasada, el borde es rojo y el fondo `error-soft`.
- **Costura de 7 puntadas:** 6 segmentos y un nudo final (la Entrega). Lo completado es una línea continua azul mezclilla; lo pendiente, punteado gris. Lleva siempre `aria-label` del tipo "3 de 7 estaciones completadas".
- **Bloque de alarma:** relleno `error` con "N ATRASADAS" en Archivo 900 y una línea secundaria ("3 salen hoy"). Sin atrasos pasa a `success-soft` con "TODO A TIEMPO".
- **Chip de estado:** palabra + forma (círculo, triángulo, cuadrado con "!", check) + color, con radio de 4 px.
- **Etiqueta del taller:** el nombre del taller en una banda `primary` con radio de 4 px, orillo turquesa de 3 px abajo y dos líneas de puntada en los extremos, en Archivo 700 mayúsculas a 0.07em, con el rol debajo. Va arriba de la barra lateral (y en la cabecera móvil). En la página de seguimiento aparece como "Hecho en" seguido del nombre del taller. Nada de píldora.
- **Barra lateral:** fondo `shell`, orillo turquesa de 3 px en su borde interior, ítem activo en `shell-raised` con un indicador turquesa de 3 px a la izquierda; los inactivos en texto secundario de la barra.
- **Enlaces:** texto `primary` en negrita con subrayado turquesa de 2 px separado 3 px del texto. El subrayado lleva el turquesa; el texto nunca.
- **Botones:** principal en azul marino, secundario con contorno azul, fantasma con borde gris, peligro en rojo (solo para acciones destructivas como "Desactivar acceso"). Estados:
  - hover: `primary-hover`;
  - foco: contorno de 3 px `focus` con 2 px de separación;
  - deshabilitado: opacidad 0.55 y cursor no permitido;
  - cargando: texto "Guardando…".
- **Campos:** 48 px de alto y borde de 1.5 px; al enfocar, borde azul. Un error marca el borde en rojo y siempre agrega un texto que dice cómo arreglarlo.
- **Avisos:** éxito en `success-soft`, error en `error-soft` con ícono rojo, información en `primary-soft`. El aviso de una acción destructiva exitosa (por ejemplo, cancelar una invitación) es informativo, no rojo.
- **Estados vacíos, de carga y de error:** son parte de cada componente, cada uno con un texto que explique qué aparecerá o qué hacer.

## Do's and Don'ts

- Do: reservar el rojo sólido solo para "Atrasada".
- Do: dar a cada estado palabra, forma y color.
- Do: usar la fuente mono para códigos OT, fechas y cantidades.
- Do: dar a los botones de operario 48 px como mínimo, en la parte de abajo.
- Do: escribir siempre colores, letras y radios como tokens (`var(--…)`), nunca valores sueltos.
- Do: usar el turquesa solo como hilo de marca (logo, orillo, ítem activo, etiqueta del taller, subrayado de enlaces) y `accent-text` cuando tenga que ser texto.
- Don't: usar verde para la marca o para botones; el verde significa "a tiempo".
- Don't: poner turquesa en chips de estado, la costura, rellenos de botón, anillos de foco o fondos de pantallas de operación, ni turquesa puro como texto sobre blanco.
- Don't: reescribir "TallerFlow" con texto en lugar del logo SVG, ni partir un titular en dos colores.
- Don't: mostrar "Completada" con color.
- Don't: poner tarjetas dentro de tarjetas, sombras en todo, degradados, morado o brillos.
- Don't: armar el panel como una grilla de 3 o 4 tarjetas de métricas iguales.
- Don't: decorar con algo que no sea la costura de 7 puntadas.

## Motion

- **Approach:** intentional.
- **Easing:** enter(ease-out) exit(ease-in) move(ease-in-out)
- **Duration:** micro(50-100ms) short(150-250ms) medium(250-400ms) long(400-700ms)
- **The one authored moment:** al confirmar el avance de una estación, la puntada correspondiente "se cose": el segmento punteado se llena de izquierda a derecha en unos 450 ms con ease-out. Con `prefers-reduced-motion`, el segmento aparece lleno sin animación.

## Logo

Un rombo tejido: una cuadrícula de 4×4 de franjas redondeadas que se entrelazan como la trama de una tela, girada 45°, en marino `primary` y turquesa `accent`. Las dos celdas vacías recortan las puntas de arriba y abajo, así que el rombo es más ancho que alto (1.37:1). Solo las esquinas exteriores son redondeadas. El logotipo es "Taller" en marino y "Flow" en turquesa (ver Typography). Sobre fondos oscuros las partes marinas pasan a blanco. Reemplaza a la etiqueta colgante de 7 puntadas del Sprint 3.

**Reglas:** se usa siempre desde los SVG, nunca redibujado ni con texto. Bajo 24 px se usa el ícono de app (rombo sobre cuadrado marino) en lugar del símbolo suelto, porque el tejido se empasta. Espacio libre alrededor: la mitad de la altura del símbolo.

### Archivos (`frontend/public/`)
- `logo-symbol.svg`: el rombo a color.
- `logo-symbol-on-dark.svg`: el rombo con las partes marinas en blanco.
- `logo-horizontal.svg`: rombo y logotipo para fondos claros.
- `logo-horizontal-on-dark.svg`: rombo y logotipo para fondos marinos u oscuros.
- `logo-monochrome.svg`: una sola tinta con `currentColor`, para incrustar en línea.
- `favicon.svg` y los PNG `favicon-16x16`, `favicon-32x32`, `apple-touch-icon` (180), `android-chrome-192x192` y `-512x512` (maskable): el rombo sobre un cuadrado marino.
- `og-image.png` (1200×630): vista previa para WhatsApp y redes.
- `frontend/resources/` (Capacitor Assets): `icon.png` (1024), `icon-foreground.png` (rombo dentro de la zona segura del 66 %), `icon-background.png` (marino) y `splash.png` (2732, logo sobre `shell`).

Los SVG se generan desde una sola definición geométrica (celda de 24, separación de 4, radio exterior de 8), así que cualquier ajuste se hace ahí y se regeneran todos.

## Decisions Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-10-17 | Sistema "Ficha viajera" creado | /design-consultation con investigación web y capturas (Katana, Aptean, Malomo, Yape), y tres propuestas independientes (Claude, Codex y un subagente de Claude). Objetivo elegido por el usuario: saber en 2 segundos qué pedido está atrasado. |
| 2026-10-17 | Marca de verde petróleo a azul mezclilla `#1F3A68` | Las tres propuestas coincidieron: el verde de marca se confundía con el verde de "a tiempo". |
| 2026-10-17 | Panel con titular de atrasadas en lugar de métricas | Cumple la promesa de los 2 segundos; patrón de "estado primero" de las páginas de seguimiento. |
| 2026-10-17 | Fichas con forma de ficha viajera y costura de 7 puntadas | Identidad propia tomada del objeto real del taller; el progreso se lee sin abrir la orden. |
| 2026-10-17 | Fuentes Archivo + Atkinson Hyperlegible Next/Mono | Rotulación textil para los titulares, y legibilidad para baja visión en el texto. |
| 2026-10-17 | Vista previa en HTML, sin maquetas de IA | La cuenta de OpenAI no tenía saldo; la vista previa usa fuentes y colores reales. |
| 2026-10-20 | Logo nuevo: rombo tejido marino `#0A346F` y turquesa `#00BFB2`; sistema "Etiqueta tejida" | Logo y colores definidos por el equipo. /design-consultation con dos voces independientes (Codex y un subagente de Claude): las tres propuestas coincidieron en mover el primario al marino del logo y dejar el turquesa como firma, nunca como estado. El PNG original se vectorizó como geometría limpia. |
| 2026-10-20 | Turquesa solo como hilo de marca, con `accent-text` `#006B64` para texto | Sobre blanco el turquesa da 2.3:1 y junto al verde de "a tiempo" 2.9:1 (1.1:1 en oscuro). Se descartaron el CTA turquesa y el nudo final turquesa de la costura porque se leerían como estado. |
| 2026-10-20 | Etiqueta tejida por taller en la barra lateral y "Hecho en" en el seguimiento | Idea del subagente: en Gamarra la etiqueta cosida es donde el taller firma su trabajo. Se descartó el corte al sesgo de 45° en la landing por acercarse a decoración. |
