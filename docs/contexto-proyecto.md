# TallerFlow - contexto del proyecto

Este documento resume el material inicial compartido en `Proyecto_inicial.pdf` y `perfiles.pdf`, y lo conecta con el Sprint 1 actual. Los PDFs son contexto de producto: no reemplazan el alcance del sprint ni cambian responsabilidades ya acordadas.

## 1. Idea central

TallerFlow es una plataforma web y movil para talleres de confeccion, enfocada en hacer visible el avance de cada orden desde que entra al taller hasta que se entrega.

La promesa principal es:

- cada orden visible;
- cada entrega bajo control;
- menos llamadas, menos reprocesos y menos busqueda manual;
- una sola vista compartida para duenio, equipo y cliente.

El problema que ataca es la desorganizacion operativa: informacion dispersa en WhatsApp, llamadas, cuadernos, Excel y conversaciones sueltas. Eso causa perdida de visibilidad, retrasos, doble registro, errores y mucho tiempo preguntando "como va el pedido".

## 2. Producto propuesto inicialmente

El producto inicial imaginado tiene tres experiencias principales:

### Panel del duenio

El duenio o administrador ve todas las ordenes en una sola vista. La propuesta visual muestra un tablero por etapas con contadores de ordenes activas, por vencer y retrasadas.

Etapas vistas en el material:

- pedido recibido;
- corte;
- confeccion o costura;
- estampado o acabado;
- control de calidad;
- listo para entregar o entrega.

El objetivo del panel es ayudar a priorizar rapido, anticipar retrasos y reducir horas de coordinacion.

### Actualizacion desde el taller

El operario o responsable de produccion puede actualizar avances desde celular. La idea inicial incluye:

- escanear una orden con QR;
- registrar cantidad terminada;
- guardar avance;
- adjuntar foto como evidencia;
- reportar un problema.

Esto reduce doble registro y permite que la informacion salga del piso de produccion casi en tiempo real.

### Seguimiento para cliente

El cliente recibe un enlace privado para consultar el estado de su orden sin instalar una app. La vista muestra avance, etapa actual y fecha estimada de entrega.

La finalidad es reducir llamadas, aumentar confianza y dar transparencia sin cargar mas trabajo administrativo al taller.

## 3. Perfiles de usuario

### Perfil 1: duenio de taller tradicional

Cliente pagador potencial. Tiene un taller de 5 a 15 operarios y trabaja con WhatsApp, llamadas, cuadernos y a veces Excel.

Problema: pierde visibilidad del avance y depende de preguntar todo el dia para saber que esta pasando.

Busca: saber que pedido esta atrasado, quien lo tiene y cuando se entregara.

Oportunidad: necesita una solucion simple, visual y facil de adoptar; no un ERP complejo.

### Perfil 2: duenio o administrador de taller en crecimiento

Cliente objetivo inicial. Coordina un taller de 10 a 30 operarios con varias ordenes en paralelo.

Problema: al crecer, se vuelve dificil controlar etapas, tiempos, retrasos y responsables.

Busca: un tablero simple para ver todas las ordenes, anticipar retrasos y tomar decisiones rapidas.

Por que es prioritario:

- tiene dolor claro y frecuente;
- ya sufre por manejar varias ordenes a la vez;
- esta mas dispuesto a pagar por visibilidad y control.

Este perfil es el mejor punto de partida para el MVP comercial.

### Perfil 3: marca o empresa que terceriza confeccion

Beneficiario externo e influenciador de adopcion. Puede ser marca de moda, mayorista, retail o cliente corporativo que envia pedidos a un taller.

Problema: debe llamar o escribir constantemente para saber como avanza su produccion.

Busca: seguimiento simple, transparente y confiable mediante enlace o vista del pedido.

Rol en TallerFlow: no necesariamente paga al inicio, pero exige transparencia, influye en la adopcion y se beneficia con el seguimiento del pedido.

## 4. Alcance real del Sprint 1

Aunque la vision inicial incluye ordenes, avances, fotos, QR y tracking para clientes, el Sprint 1 no construye todo eso todavia.

El Sprint 1 se enfoca en cimientos seguros:

- base de datos PostgreSQL con migraciones;
- usuarios, talleres, membresias y roles;
- autenticacion segura;
- sesiones;
- aislamiento entre talleres;
- login frontend;
- entorno Docker reproducible;
- CI basico;
- un OWNER creado por bootstrap que puede iniciar sesion, cerrar sesion y ver su taller.

Por eso, las ordenes de produccion, avances, evidencias, QR, clientes externos y tracking quedan para fases posteriores.

## 5. Como se conecta la vision con Sprint 1

La vision de producto necesita que cada taller vea solo su informacion. Antes de crear ordenes, avances o vistas para clientes, el sistema debe resolver identidad, roles y aislamiento.

Sprint 1 responde a esas preguntas base:

- quien es el usuario;
- a que taller pertenece;
- que rol tiene;
- como inicia sesion de forma segura;
- como se evita que un taller vea datos de otro;
- como se levanta el sistema desde cero de forma reproducible.

Sin esos cimientos, las futuras funcionalidades de trazabilidad tendrian riesgo de seguridad y desorden tecnico.

## 6. Responsabilidades por persona

### Jose

Se encarga del backend de autenticacion e integracion API:

- plataforma Go;
- health checks;
- Argon2id;
- sesiones opacas;
- login, logout y restauracion de sesion;
- CSRF, cookies y rate limit;
- bootstrap del primer OWNER.

### Lucero

Se encarga del frontend base y experiencia de login:

- Vue 3, Vite y TypeScript;
- router, estado y cliente HTTP;
- sistema visual de TallerFlow;
- login responsive y accesible;
- restauracion de sesion;
- vista de cambio obligatorio de contrasena.

El link de Stitch sirve como referencia visual para ella y para alinear el estilo del producto.

### Michael

Se encarga de multi-tenancy, talleres y roles:

- conexion de backend a PostgreSQL;
- TenantRunner;
- principal autenticado;
- resolucion de taller activo;
- roles y membresias;
- endpoints `/me` y taller actual;
- pruebas de aislamiento entre talleres.

### Stephano

Stephano se encarga de datos, contenedores e infraestructura de desarrollo/CI. Su trabajo es principalmente backend de plataforma, base de datos e infraestructura; no frontend visual.

Le toca:

- `database/migrations/`;
- `infra/`;
- `compose.yaml`;
- `compose.production.yaml`;
- `.env.example`;
- `.github/workflows/`;
- `backend/Dockerfile`;
- `frontend/Dockerfile`;
- `README.md`.

Entregables principales:

- PostgreSQL y Flyway;
- roles separados de propietario, migracion y aplicacion;
- migraciones iniciales;
- constraints, indices, grants y RLS;
- Docker Compose con servicios saludables;
- imagenes multi-stage sin root;
- Caddy same-origin;
- CI de datos, backend y frontend;
- verificacion desde volumenes vacios.

Stephano no deberia tocar:

- reglas de autenticacion;
- casos de uso de talleres;
- componentes visuales;
- pantallas frontend, salvo que sea necesario para Docker/build.

La rama correcta para Stephano deberia ser:

```text
feat/s1-stephano-data-platform
```

## 7. Lectura final de coherencia

Hay hilacion entre la propuesta inicial y el Sprint 1:

- La propuesta inicial vende trazabilidad y control operativo.
- Los perfiles muestran que el usuario prioritario es el duenio/administrador de taller en crecimiento.
- El Sprint 1 no construye aun la trazabilidad completa, pero prepara la base segura para construirla despues.
- Lo que le toca a Stephano es clave porque define la base donde viviran ordenes, avances, clientes y aislamiento por taller en fases posteriores.

En simple: TallerFlow quiere llegar a "cada orden visible"; Sprint 1 construye primero "cada taller seguro, cada usuario identificado y cada entorno reproducible".
