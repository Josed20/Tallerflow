# Propuesta de logo TallerFlow

El símbolo une una forma de T/F con una banda plegada que avanza: una referencia discreta al textil y al flujo de producción. Se diseñó como vector plano para que funcione desde un favicon de 16 px hasta una presentación.

## Archivos

- `frontend/public/brand/logo.svg`: firma horizontal sobre fondo claro.
- `frontend/public/brand/logo-on-dark.svg`: firma horizontal sobre fondo oscuro.
- `frontend/public/brand/symbol.svg` y `symbol-on-dark.svg`: símbolo sin texto y con fondo transparente.
- `frontend/public/brand/favicon.svg`: símbolo con base oscura para pestañas.
- `docs/brand/preview.svg` y `preview.png`: lámina comparativa para revisar la propuesta.

Los SVG son los originales escalables. Las exportaciones PNG son para presentaciones y contextos que no admiten SVG. Mantener espacio libre alrededor del símbolo y no aplicar sombras, degradados ni recoloraciones aisladas.

## Uso en la aplicación

`BrandLogo.vue` selecciona la firma horizontal o el símbolo y su versión clara u oscura. La autenticación, el portal del cliente y la página 404 usan la firma sobre fondo claro; el menú interno usa la firma clara sobre fondo oscuro. `frontend/index.html` declara el favicon SVG con respaldo PNG y el icono de 512 px para dispositivos.

La marca comparte verde petróleo (`#00685f`), menta (`#53c7b2`), fondo de navegación (`#143e40`) y tinta (`#0b1c30`). Los tres perfiles internos conservan esta identidad y los colores semánticos de las órdenes. Solo cambia el acento contextual: propietario menta, administrador azul y operario ámbar. El acento aparece en la navegación activa, el contexto del taller y el avatar; no se usa para alterar el significado de éxito, riesgo o atraso.
