---
# gstack: design-md-format=spec
name: TallerFlow
description: Ficha viajera. Claridad de letrero de taller en azul mezclilla; el único color que grita es el de un pedido atrasado.
colors:
  primary: "#1F3A68"
  primary-hover: "#2A4C85"
  primary-soft: "#E4EBF6"
  on-primary: "#FFFFFF"
  surface: "#FFFFFF"
  background: "#F3F5F9"
  text: "#16233D"
  text-muted: "#556074"
  border: "#D5DCE6"
  focus: "#1F3A68"
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
---

# TallerFlow

## Overview

**Creative North Star:** "Ficha viajera": industrial y utilitario con alma textil. La app toma sus formas de los objetos reales del taller (la ficha que viaja con cada lote, la costura, la etiqueta colgante) y se lee con la claridad de un letrero desde el otro lado del pasillo.
**Lo que todo usuario debe recordar:** "Abro el celular y en 2 segundos sé qué pedido está atrasado." Cada decisión de este documento sirve a esa frase.
**Product context:** trazabilidad de órdenes para talleres de confección de Gamarra (Lima). Usuarios: dueño o administrador de taller de 10–30 operarios (cliente objetivo), dueño de taller tradicional con poca experiencia digital, operarios en planta con celulares Android de 360 px y la marca cliente que sigue su pedido por un enlace público.
**Mode per surface:**
- Panel, tablero, detalle de orden, equipo, clientes: Operate.
- Página pública de seguimiento (`/seguimiento/:token`): Read, con el estado primero.
- Landing de presentación: Persuade (debe alinearse a este sistema).
**Reference sites (investigación del 2026-10-17):** https://katanamrp.com, https://www.aptean.com/en-US/solutions/mes/products/apparel-shop-floor-control-exenta, https://gomalomo.com/order-tracking-pages-that-inspire, https://www.yape.com.pe. Ningún producto del rubro textil tiene voz visual propia: el software de planta es corporativo y gris, o un SaaS genérico.
**Vista previa aprobada:** https://claude.ai/artifact/4wc2hv9KE4hf1WPQunj3i8
**Key characteristics:**
- La marca calla (azul mezclilla y grises) para que el rojo de "Atrasada" sea lo único que grita.
- El panel abre con un titular, no con una grilla de métricas.
- Las órdenes tienen forma de ficha viajera, con una costura de 7 puntadas que muestra el avance.
- Cada estado se reconoce por palabra, forma y color.

## Colors

**Strategy:** Restrained. Un color de marca más neutros derivados de él; el color fuerte se reserva para el estado de las órdenes.
**Light or dark:** claro por defecto. Se usa bajo luz fluorescente de taller y de día, en celulares baratos. El tema oscuro existe para el dueño que revisa de noche.

- **Azul mezclilla (`primary`)** es la marca: logo, botones principales, foco, progreso y la costura completada. Es la tela de Gamarra, y al no ser verde deja el verde libre para significar "a tiempo".
- **Semáforo**, el corazón del sistema:
  - A tiempo: texto `success` sobre `success-soft`, con círculo.
  - En riesgo (vence en 48 h): relleno `warning` con texto `on-warning`, con triángulo.
  - Atrasada: relleno sólido `error` con texto blanco, con cuadrado y "!". Es el único relleno saturado de la interfaz; si algo más usa rojo sólido, el rojo deja de avisar.
  - Completada: grafito `completed` sobre `completed-soft`, con check. Pierde color a propósito: lo terminado se guarda y no compite con lo urgente.
- **Accesibilidad:** el rojo y el verde tienen casi la misma luminancia (1.12:1), así que una persona daltónica no los distingue por color. Por eso ningún estado depende solo del color. Todos los pares de texto superan 4.5:1 y el foco tiene 10:1 sobre el fondo.
- **Tema oscuro** (redefine tokens, no invierte): fondo `#0F1624`, superficie `#172236`, texto `#E7EDF6`, secundario `#A1AEC2`, borde `#2A3850`, marca `#8FB0E8` con texto `#0B1424` encima, a tiempo `#5DD39A` sobre `#123222`, atrasada mantiene `#C8102E` con texto blanco, completada `#9A9DB4` sobre `#222C3E`. El ámbar de riesgo no cambia.
- **Reemplaza** al verde petróleo `#00685f` anterior, al foco `#6be7d6` (1.5:1, no cumplía), y unifica los dos rojos (`--tf-error` y `--tf-danger`) en `error`.

## Typography

Tres familias gratuitas de Google Fonts, verificadas contra la API de Google Fonts el 2026-10-17:
`https://fonts.googleapis.com/css2?family=Archivo:wdth,wght@62..125,100..900&family=Atkinson+Hyperlegible+Mono:wght@200..800&family=Atkinson+Hyperlegible+Next:wght@200..800&display=swap`

- **Archivo, ancho expandido (`wdth` 125), 800–900:** titulares, el bloque de atrasadas y cifras grandes. Ancha y firme, como la rotulación de un fardo. Usar `font-variation-settings: "wdth" 125`, porque `font-stretch` no siempre responde con la carga de Google Fonts.
- **Archivo, ancho normal, 700, mayúsculas con 0.07em de espaciado:** etiquetas cortas ("Carga por estación"). El espaciado en mayúsculas pequeñas no se quita.
- **Atkinson Hyperlegible Next:** todo el texto y los botones. Diseñada para baja visión: distingue 0/O y 1/l/I, clave para operarios apurados y dueños con poca confianza digital.
- **Atkinson Hyperlegible Mono:** códigos `OT-000003`, fechas, horas y cantidades, con números del mismo ancho.
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
- **Botones:** principal en azul mezclilla, secundario con contorno azul, fantasma con borde gris, peligro en rojo (solo para acciones destructivas como "Desactivar acceso"). Estados:
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
- Don't: usar verde para la marca o para botones; el verde significa "a tiempo".
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

Una etiqueta colgante con ojal y la esquina recortada de la ficha, cruzada por una costura de 7 puntadas: 6 trazos y un punto lleno final (la Entrega). El logotipo "TallerFlow" va en Archivo expandida 800. Funciona en un solo color (tinta sobre claro, blanco sobre azul mezclilla) y se imprime en la ficha viajera. Se reconoce hasta en 16 px como favicon.

### Paquete de activos entregado (Sprint 3 — Lucero)
- **Vectores en `frontend/public/`:**
  - `logo-symbol.svg`: Isotipo de la etiqueta colgante con las 7 puntadas.
  - `logo-horizontal.svg`: Versión horizontal completa con wordmark para fondos claros.
  - `logo-on-denim.svg`: Versión completa en blanco sobre fondo azul mezclilla `#1F3A68`.
  - `logo-monochrome.svg`: Versión monocromática con `currentColor`.
- **Favicons y Web Manifest:**
  - `favicon.svg`, `favicon-16x16.png`, `favicon-32x32.png`, `apple-touch-icon.png` (180×180).
  - `android-chrome-192x192.png`, `android-chrome-512x512.png` (íconos maskable).
  - `manifest.webmanifest`.
- **Redes sociales y mensajería:**
  - `og-image.png` (1200×630): Vista previa para compartir enlaces en WhatsApp, Facebook y Twitter con etiquetas Open Graph en `index.html`.
- **App móvil Android (`frontend/resources/` para Capacitor Assets):**
  - `icon.png` (1024×1024): Ícono maestro de la aplicación.
  - `icon-foreground.png` e `icon-background.png`: Ícono adaptativo de Android.
  - `splash.png` (2732×2732): Pantalla de inicio con logo centrado sobre denim oscuro `#16294A`.

## Decisions Log
| Date | Decision | Rationale |
|------|----------|-----------|
| 2026-10-17 | Sistema "Ficha viajera" creado | /design-consultation con investigación web y capturas (Katana, Aptean, Malomo, Yape), y tres propuestas independientes (Claude, Codex y un subagente de Claude). Objetivo elegido por el usuario: saber en 2 segundos qué pedido está atrasado. |
| 2026-10-17 | Marca de verde petróleo a azul mezclilla `#1F3A68` | Las tres propuestas coincidieron: el verde de marca se confundía con el verde de "a tiempo". |
| 2026-10-17 | Panel con titular de atrasadas en lugar de métricas | Cumple la promesa de los 2 segundos; patrón de "estado primero" de las páginas de seguimiento. |
| 2026-10-17 | Fichas con forma de ficha viajera y costura de 7 puntadas | Identidad propia tomada del objeto real del taller; el progreso se lee sin abrir la orden. |
| 2026-10-17 | Fuentes Archivo + Atkinson Hyperlegible Next/Mono | Rotulación textil para los titulares, y legibilidad para baja visión en el texto. |
| 2026-10-17 | Vista previa en HTML, sin maquetas de IA | La cuenta de OpenAI no tenía saldo; la vista previa usa fuentes y colores reales. |
