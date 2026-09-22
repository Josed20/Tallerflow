# TallerFlow — Sprint 1: Cimientos seguros

**Estado:** Propuesto para revisión y ejecución

**Duración:** 10 días hábiles

**Meta:** un OWNER provisionado por bootstrap inicia y cierra sesión, consulta su taller y queda aislado de otros talleres.

**Fuentes de verdad:**

- Arquitectura: `docs/superpowers/specs/2026-09-21-tallerflow-mvp-design.md`
- Hoja de ruta: `docs/superpowers/plans/2026-09-21-tallerflow-mvp-implementation.md`
- Plan ejecutable: `docs/superpowers/plans/2026-09-21-tallerflow-phase-1-foundations.md`

## 1. Alcance

Incluye:

- PostgreSQL 18, Flyway y RLS inicial.
- Plataforma Go, configuración, errores y health checks.
- Argon2id, sesiones opacas, cookies, CSRF y rate limit.
- Talleres, membresías, roles y principal autenticado.
- Bootstrap del primer OWNER.
- Vue 3 + TypeScript, sistema visual, login y restauración de sesión.
- Docker Compose, Caddy, CI y recorrido E2E.

No incluye clientes, órdenes, avances, tracking externo, cargas de imágenes ni despliegue en VPS. Esos elementos pertenecen a fases posteriores.

## 2. Estrategia Git

Las cuatro ramas parten del mismo `main`:

```powershell
git switch main
git pull --ff-only
git switch -c feat/s1-<persona>-<area>
```

Ramas:

| Persona | Rama |
|---|---|
| José | `feat/s1-jose-auth-api` |
| Lucero | `feat/s1-lucero-frontend-shell` |
| Michael | `feat/s1-michael-tenancy-team` |
| Stephano | `feat/s1-stephano-data-platform` |

Reglas:

- No trabajar directamente en `main`.
- No mezclar dos entregables independientes en un commit.
- Actualizar la rama con `main` antes de abrir o actualizar PR.
- No reescribir migraciones que ya hayan sido fusionadas.
- Stephano es propietario de la numeración Flyway.
- José es propietario del router central y composición de dependencias.
- Los módulos exportan `RegisterRoutes`; sus autores no cablean rutas globales.
- Los cambios de DTO se reflejan primero en OpenAPI.

## 3. José — API, autenticación e integración

**Objetivo:** entregar el núcleo Go y el flujo seguro de autenticación.

**Tareas del plan ejecutable:** 1, 4, 5, 7 y coordinación de 11.

**Archivos propios:**

```text
backend/cmd/api/
backend/cmd/bootstrap/
backend/internal/auth/
backend/internal/audit/
backend/platform/config/
backend/platform/httpx/
backend/platform/security/
```

**Entregables:**

1. Módulo Go, configuración segura y request IDs.
2. `/health/live` y composición del router.
3. PasswordHasher Argon2id con formato PHC.
4. Sesiones opacas cuyo valor raw nunca entra a PostgreSQL.
5. Login, logout y restauración de sesión.
6. Cambio obligatorio de contraseña con rotación.
7. Cookie segura y CSRF.
8. Rate limit y respuesta genérica de credenciales.
9. Bootstrap transaccional del OWNER.
10. Rama de integración y recorrido E2E.

**PR sugeridos:**

- `feat(platform): bootstrap Go API and health endpoint`
- `feat(auth): add Argon2id credentials and opaque sessions`
- `feat(auth): expose secure cookie session flow`
- `feat(auth): provision initial workshop owner`

**José no modifica:**

- Migraciones numeradas sin revisión de Stephano.
- Componentes visuales de Lucero.
- Implementación del tenant runner de Michael.

**Alcance de Codex:** la implementación asistida en este repositorio se limitará inicialmente a las tareas de José. El trabajo de las demás personas permanece documentado para que se desarrolle en sus ramas.

## 4. Lucero — Vue, diseño y experiencia de login

**Objetivo:** entregar una base visual profesional y un login accesible sin almacenar tokens.

**Tareas del plan ejecutable:** 8 y 9.

**Archivos propios:**

```text
frontend/src/app/
frontend/src/modules/auth/
frontend/src/modules/session/
frontend/src/components/
frontend/src/layouts/
frontend/src/styles/
```

**Entregables:**

1. Vue 3, Vite y TypeScript.
2. Router, Pinia, TanStack Query, Tailwind y Reka UI.
3. Tokens visuales de TallerFlow.
4. Botón, campo, alerta y layout de autenticación.
5. Cliente HTTP same-origin.
6. Store de sesión solo en memoria.
7. Login responsive, validado y accesible.
8. Vista de cambio obligatorio de contraseña.
9. Guard de rutas y restauración antes de navegar.
10. Pruebas unitarias y build.

**PR sugeridos:**

- `feat(frontend): add Vue TypeScript design foundation`
- `feat(frontend-auth): add secure login and session restore`

**Lucero no modifica:**

- Cookies, CSRF o rate limit del backend.
- Migraciones.
- Contratos sin coordinar con José y Michael.

## 5. Michael — Multi-tenancy, talleres y roles

**Objetivo:** garantizar que la aplicación siempre opere con un principal y taller válidos.

**Tareas del plan ejecutable:** 3 y 6.

**Archivos propios:**

```text
backend/platform/database/
backend/internal/workshops/
backend/platform/httpx/principal.go
```

**Entregables:**

1. Conexión GORM a PostgreSQL.
2. TenantRunner con `SET LOCAL app.workshop_id`.
3. Pruebas de ausencia de contexto y aislamiento cruzado.
4. Dominio de roles y membresías.
5. Resolución de un único taller activo.
6. Principal para handlers.
7. `GET /api/v1/me`.
8. `GET /api/v1/workshops/current`.
9. Middleware de roles.

**PR sugeridos:**

- `feat(database): enforce tenant-scoped transactions`
- `feat(workshops): resolve tenant principal and roles`

**Michael no modifica:**

- Router central.
- PasswordHasher y cookies.
- Números de migración.
- Componentes Vue.

## 6. Stephano — Datos, contenedores y CI

**Objetivo:** producir un entorno reproducible y una base segura.

**Tareas del plan ejecutable:** 2 y 10.

**Archivos propios:**

```text
database/migrations/
infra/
compose.yaml
compose.production.yaml
.env.example
.github/workflows/
backend/Dockerfile
frontend/Dockerfile
README.md
```

**Entregables:**

1. PostgreSQL 18.6 y Flyway 13.7.0.
2. Roles separados de propietario, migración y aplicación.
3. Migraciones V1 y V2.
4. Checks, índices, grants y RLS.
5. Compose con dependencias saludables.
6. Imágenes multi-stage sin root.
7. Caddy same-origin.
8. CI de datos, backend y frontend.
9. Verificación desde volúmenes vacíos.

**PR sugeridos:**

- `feat(database): add identity schema and tenant policies`
- `ci: integrate Docker Caddy and validation pipeline`

**Stephano no modifica:**

- Reglas de autenticación.
- Casos de uso de talleres.
- Componentes visuales.

## 7. Oleadas

### Días 1–2: contratos y esqueletos

| Persona | Resultado |
|---|---|
| José | Go compila, config y live health |
| Lucero | Vue compila, tokens y componentes base |
| Michael | Interfaces de TenantRunner y Principal acordadas |
| Stephano | PostgreSQL, Flyway, V1 y V2 |

Control:

```text
Flyway migra una base vacía
Backend y frontend compilan
Interfaces de auth/tenant congeladas
Nadie implementa clientes u órdenes
```

### Días 3–5: seguridad

| Persona | Resultado |
|---|---|
| José | Argon2id, sesiones y repositorios |
| Lucero | cliente HTTP, store y formulario de login |
| Michael | TenantRunner y pruebas RLS |
| Stephano | grants, roles y Compose integrado |

Control:

```text
Token raw no aparece en DB
RLS niega ausencia de contexto
Frontend no usa localStorage
Login inválido tiene mensaje genérico
```

### Días 6–8: integración de identidad

| Persona | Resultado |
|---|---|
| José | handlers, CSRF, rate limit y bootstrap |
| Lucero | login conectado y guardas |
| Michael | principal, /me y taller actual |
| Stephano | Caddy, Dockerfiles y CI |

Control:

```text
OWNER inicia sesión
/me devuelve taller y rol
Logout revoca la sesión
CSRF ausente produce 403
```

### Días 9–10: estabilización

- Fusionar ramas en el orden acordado.
- Crear `feat/s1-integration-foundations` desde `main` actualizado.
- Ejecutar E2E y pruebas negativas.
- Probar 360 px, teclado y sesión expirada.
- Levantar desde volúmenes vacíos.
- Corregir defectos críticos y altos.
- Actualizar runbooks.

## 8. Dependencias y orden de merge

| Entregable | Depende de | Bloquea |
|---|---|---|
| Migraciones | Spec aprobada | repositorios auth y talleres |
| Backend base | ninguna | handlers y Compose |
| Password/session service | backend base + tablas | login |
| TenantRunner | tablas + rol app | principal y RLS |
| Vue shell | contrato visual | login |
| Principal | sesión + TenantRunner | `/me` |
| Caddy/Compose | builds estables | E2E |
| E2E | todos los anteriores | cierre |

Orden:

1. Stephano: datos.
2. José: plataforma y servicios auth.
3. Michael: tenancy y talleres.
4. Lucero: frontend.
5. Stephano: infraestructura integrada.
6. José: integración final.

## 9. Revisión de PR

| Autor | Revisor principal | Foco |
|---|---|---|
| José | Michael | seguridad, interfaces y testabilidad |
| Lucero | José | contrato, errores, sesión y accesibilidad |
| Michael | José | RLS, principal, roles y transacciones |
| Stephano | Michael | SQL, grants, Compose y reproducibilidad |

Stephano revisa cualquier columna o variable nueva. Lucero revisa cambios de DTO consumidos por Vue.

Checklist del autor:

```markdown
- [ ] La rama parte de main actualizado.
- [ ] El cambio pertenece al Sprint 1.
- [ ] Escribí primero la prueba relevante.
- [ ] Ejecuté los comandos del plan.
- [ ] No agregué secretos.
- [ ] No usé AutoMigrate ni JWT de navegador.
- [ ] Las consultas privadas tienen contexto de taller.
- [ ] Actualicé OpenAPI o documentación si cambió una interfaz.
- [ ] El CI está verde.
```

## 10. Definición de terminado

1. `docker compose up --build` funciona desde limpio.
2. Flyway valida y migra sin error.
3. Bootstrap crea un OWNER exactamente una vez.
4. El primer login obliga a cambiar la contraseña y rota la sesión.
5. OWNER ve su taller.
6. Logout y expiración invalidan la sesión.
7. Mutación sin CSRF recibe `403`.
8. Intentos excesivos reciben `429`.
9. RLS bloquea otro taller y contexto ausente.
10. Vue no guarda credenciales o tokens en almacenamiento web.
11. Login funciona con teclado y a 360 px.
12. Backend, frontend, datos y E2E pasan en CI.
13. README y runbooks contienen comandos comprobados.

## 11. Demo

1. Stephano levanta PostgreSQL y Flyway desde cero.
2. José ejecuta bootstrap sin mostrar la contraseña.
3. Lucero abre el login en escritorio y móvil.
4. José inicia sesión.
5. Michael muestra `/me`, rol y taller activo.
6. José demuestra error genérico, CSRF y logout.
7. Michael ejecuta el test de aislamiento RLS.
8. Stephano muestra CI verde.

El Sprint termina únicamente después de repetir la demo desde un entorno limpio.
