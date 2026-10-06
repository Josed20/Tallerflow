# Auditoría de despliegue — TallerFlow

**Fecha:** 26 de septiembre de 2026
**Alcance evaluado:** rama entregable `main`, commit `477aed19cdd68417c776ae732042b3efbae8281d`.
**Método:** inspección de Git, archivos versionados y configuración; análisis estático de secretos de alta confianza en todos los refs locales; verificación local de Docker cuando hubo artefactos que probar.

> **Nota de alcance importante.** La rama `main` sólo tiene seis archivos versionados (README y documentación) y no contiene el código ni archivos de despliegue. Existen worktrees no fusionados, entre ellos `codex/s1-integration`, con una implementación candidata. No se les otorga puntaje: un artefacto fuera de `main` no es desplegable como entrega oficial. Sus hallazgos técnicos se documentan al final como evidencia adicional.

## 📊 TABLA DE RESULTADOS

| Etapa | # | Ítem | Estado | Observación |
|---|---:|---|---|---|
| 1 | 1.1 | Repositorio Git con historial | ✅ CUMPLE | Repositorio válido con siete commits en `main`; último: `477aed1`. |
| 1 | 1.2 | Ramas `main` y `develop` separadas | ❌ NO CUMPLE | Existe `main`; no existe una rama local ni remota `develop`. |
| 1 | 1.3 | Ramas `feature/nombre` | ✅ CUMPLE | Existen, por ejemplo, `feat/s1-jose-auth-api` y `feat/s1-stephano-data-platform`. |
| 1 | 1.4 | Prefijos semánticos en commits | ⚠️ CUMPLE PARCIALMENTE | Los commits recientes usan `docs:` y `chore:`, pero el commit inicial no tiene prefijo y no hay convención documentada/enforced. |
| 1 | 1.5 | Tag semántico `vX.Y.Z` | ❌ NO CUMPLE | `git tag --list` no devuelve tags. |
| 1 | 1.6 | `.gitignore` excluye `node_modules`, `.env`, build y logs | ❌ NO CUMPLE | En `main`, `.gitignore` sólo contiene `.worktrees/`. |
| 1 | 1.7 | Sin secretos o contraseñas en historial | ⚠️ CUMPLE PARCIALMENTE | No se hallaron patrones de tokens de alta confianza (AWS, GitHub, claves privadas, Google, Slack) en los refs, pero no existe escáner dedicado ni regla preventiva; no puede certificarse ausencia total. |
| 1 | 1.8 | `.github/workflows/` con CI/CD | ❌ NO CUMPLE | No está versionada en `main`. |
| 1 | 1.9 | CI ejecuta pruebas antes de fusionar | ❌ NO CUMPLE | No existe pipeline en la rama entregable. |
| 1 | 1.10 | CI usa `npm ci` | ❌ NO CUMPLE | No existe pipeline ni proyecto Node versionado en `main`. |
| 2 | 2.1 | Dockerfile backend | ❌ NO CUMPLE | No hay Dockerfile en `main`. |
| 2 | 2.2 | Imagen base específica y ligera | ❌ NO CUMPLE | No hay imagen definida para evaluar. |
| 2 | 2.3 | Dependencias antes de copiar código | ❌ NO CUMPLE | No hay Dockerfile. |
| 2 | 2.4 | Contenedor no root (`USER`) | ❌ NO CUMPLE | No hay Dockerfile. |
| 2 | 2.5 | `EXPOSE` del puerto de aplicación | ❌ NO CUMPLE | No hay Dockerfile. |
| 2 | 2.6 | `.dockerignore` excluye dependencias, `.env`, `.git` | ❌ NO CUMPLE | No hay `.dockerignore` en `main`. |
| 2 | 2.7 | Multi-stage build si compila | ❌ NO CUMPLE | La aplicación compilable y su Dockerfile no están integrados en `main`. |
| 2 | 2.8 | `docker build -t nombre:tag .` construye | ❌ NO CUMPLE | No hay contexto/Dockerfile en la rama evaluada. |
| 2 | 2.9 | Contenedor inicia y responde HTTP | ❌ NO CUMPLE | No existe contenedor de aplicación ejecutable en `main`. |
| 2 | 2.10 | Endpoint `/health` retorna 200 | ❌ NO CUMPLE | No existe backend integrado en `main`. |
| 3 | 3.1 | `docker-compose.yml` o equivalente | ❌ NO CUMPLE | No hay archivo Compose en `main`. |
| 3 | 3.2 | Backend y BD en servicios separados | ❌ NO CUMPLE | No hay definición de servicios. |
| 3 | 3.3 | Volumen nombrado para BD | ❌ NO CUMPLE | No hay definición de BD/volúmenes. |
| 3 | 3.4 | Healthcheck de BD | ❌ NO CUMPLE | No hay servicio de BD. |
| 3 | 3.5 | `depends_on.condition: service_healthy` | ❌ NO CUMPLE | No hay servicios Compose. |
| 3 | 3.6 | App usa nombre del servicio de BD, no `localhost` | ❌ NO CUMPLE | No hay configuración de aplicación ni de BD. |
| 3 | 3.7 | Variables `${VARIABLE}` desde `.env` | ❌ NO CUMPLE | No hay Compose ni `.env.example` en `main`. |
| 3 | 3.8 | `docker compose up -d --build` funciona | ❌ NO CUMPLE | No hay Compose ejecutable. |
| 3 | 3.9 | Persistencia tras `down` y `up` | ❌ NO CUMPLE | No hay volumen ni stack que verificar. |
| 3 | 3.10 | Logs mediante `docker compose logs -f` | ❌ NO CUMPLE | No hay stack Compose. |
| 4 | 4.1 | `.env.example` completo y sin secretos reales | ❌ NO CUMPLE | No existe en `main`. |
| 4 | 4.2 | `.env` real no está versionado | ❌ NO CUMPLE | Aunque no hay `.env` versionado, `.gitignore` no lo protege; el control exigido no está implementado. |
| 4 | 4.3 | Sin secretos hardcodeados | ❌ NO CUMPLE | No hay aplicación/configuración integrada que permita demostrar el control; tampoco hay política o escaneo de secretos. |
| 4 | 4.4 | Configuración por ambiente mediante variables | ❌ NO CUMPLE | No hay configuración de aplicación en la rama entregable. |
| 4 | 4.5 | Logs a stdout/stderr | ❌ NO CUMPLE | No existe proceso de aplicación integrado que pueda verificarse. |
| 4 | 4.6 | Migraciones versionadas | ❌ NO CUMPLE | No hay directorio ni scripts de migración en `main`. |
| 4 | 4.7 | App espera migraciones exitosas | ❌ NO CUMPLE | No existe Flyway ni orquestación de inicio. |
| 4 | 4.8 | Misma imagen para DEV, QA y PROD | ❌ NO CUMPLE | No hay imagen Docker. |
| 4 | 4.9 | `/health` valida conexión real a BD | ❌ NO CUMPLE | No hay backend ni conexión a BD integrados. |
| 4 | 4.10 | Aplicación de los 12 factores | ❌ NO CUMPLE | No hay implementación desplegable en `main` para evidenciar los factores. |
| 5 | 5.1 | Dockerfile independiente de frontend | ❌ NO CUMPLE | El frontend está contemplado por el proyecto, pero no está integrado en `main`. |
| 5 | 5.2 | Build multi-stage + runtime para frontend | ❌ NO CUMPLE | No hay Dockerfile de frontend en la rama evaluada. |
| 5 | 5.3 | URL de API inyectada, no hardcodeada | ❌ NO CUMPLE | No hay fuente ni configuración de frontend en `main`. |
| 5 | 5.4 | Servicio propio de frontend en Compose | ❌ NO CUMPLE | No hay archivo Compose. |
| 5 | 5.5 | Frontend construye y sirve desde contenedor | ❌ NO CUMPLE | No existe frontend integrado ni imagen que ejecutar. |

## 🏷️ NIVEL DE MADUREZ

**Puntaje: 2 / 45 puntos.** Se asigna un punto únicamente a cada ítem marcado **✅ CUMPLE**; los estados parciales no suman el punto completo.

**🔴 NIVEL 0: Solo código. Sin despliegue.** En realidad, la rama que se podría entregar hoy contiene documentación y gestión inicial de Git, no el código de aplicación ni sus activos de despliegue. Por tanto, ni siquiera puede iniciarse el ciclo Docker/Compose desde `main`.

## 🚨 HALLAZGOS CRÍTICOS (bloqueantes)

### 1.2 — Falta la rama `develop`

- **Qué falta:** una rama de integración separada de `main` y una regla de protección/PR para fusionar hacia ella y luego a `main`.
- **Por qué es crítico:** el principio de Git Flow de la semana 02 separa integración y entrega estable; sin ello no hay una promoción controlada de artefactos.
- **Cómo corregirlo:**

  ```powershell
  git switch main
  git switch -c develop
  git push -u origin develop
  ```

  Configure PR obligatorios y CI requerido para `develop` y `main` en el proveedor Git.

### 1.5 — No hay versión liberable trazable

- **Qué falta:** un tag semántico inmutable asociado a una versión probada.
- **Por qué es crítico:** el curso exige trazabilidad entre fuente, imagen y despliegue; sin tag no hay artefacto reproducible al que volver.
- **Cómo corregirlo:** cuando `main` contenga una entrega validada:

  ```powershell
  git tag -a v0.1.0 -m "Primera entrega desplegable"
  git push origin v0.1.0
  ```

### 1.6, 4.2 y 4.3 — No existe una barrera efectiva de secretos/configuración local

- **Qué falta:** exclusiones para `.env`, `node_modules/`, artefactos de build y logs, una plantilla `.env.example`, y escaneo de secretos en pre-commit/CI.
- **Por qué es crítico:** configuración y credenciales son factores externos (Twelve-Factor, Factor III); versionarlas o dejarlas sin protección permite fugas y hace los entornos no reproducibles.
- **Cómo corregirlo:**

  ```gitignore
  # .gitignore
  .env
  .env.*
  !.env.example
  node_modules/
  dist/
  build/
  *.log
  ```

  Agregue `.env.example` sólo con nombres/valores evidentemente ficticios e incluya en CI, por ejemplo, `gitleaks detect --source . --no-git` y `gitleaks detect --log-opts="--all"`.

### 1.8–1.10 — No hay CI que valide el artefacto antes de integrar

- **Qué falta:** workflow versionado en `.github/workflows/ci.yml` que corra en PR y antes de la promoción.
- **Por qué es crítico:** CI es la puerta de calidad de la semana 02; sin pruebas y build automatizados, `main` puede contener un artefacto no ejecutable.
- **Cómo corregirlo:** cree el workflow, incluyendo pruebas y build de imágenes. Para un frontend Node, use instalación determinista:

  ```yaml
  - uses: actions/setup-node@v4
  - working-directory: frontend
    run: npm ci
  - working-directory: frontend
    run: npm test -- --run
  - run: docker build -t tallerflow-backend:${{ github.sha }} backend
  ```

### 2.1–2.10 — No hay empaquetado Docker del backend en `main`

- **Qué falta:** Dockerfile, `.dockerignore`, build repetible, usuario no root, puerto y healthcheck del backend.
- **Por qué es crítico:** la semana 03 convierte el código en una unidad portable y verificable. Sin imagen no hay contenedor que promover a QA/producción.
- **Cómo corregirlo:** fusione una implementación que incluya esos artefactos y verifique desde la raíz:

  ```powershell
  docker build -t tallerflow-backend:v0.1.0 ./backend
  docker run --rm -p 8080:8080 --env-file .env tallerflow-backend:v0.1.0
  Invoke-WebRequest http://localhost:8080/health/live
  ```

### 3.1–3.10 — No existe orquestación reproducible

- **Qué falta:** archivo Compose con backend, BD, volumen nombrado, healthcheck y dependencias condicionadas.
- **Por qué es crítico:** la semana 04 requiere levantar el sistema completo de forma determinista; sin ello no puede probarse la conectividad ni la persistencia de la BD.
- **Cómo corregirlo:** versionar `compose.yaml`, usar un volumen nombrado, `DB_HOST=postgres` y condiciones `service_healthy`/`service_completed_successfully`; luego validar:

  ```powershell
  docker compose up -d --build
  docker compose ps
  docker compose logs --follow
  ```

### 4.1 y 4.4–4.10 — Falta la base Twelve-Factor y de migraciones en la entrega

- **Qué falta:** configuración externa, logs stdout/stderr, migraciones versionadas, espera de migración y readiness real contra la BD.
- **Por qué es crítico:** las semanas 04–05 exigen que la misma imagen cambie de ambiente sólo por configuración y que no anuncie disponibilidad antes de sus dependencias.
- **Cómo corregirlo:** incluya `.env.example`, migraciones `database/migrations/V1__...sql`, un servicio migrador, y un endpoint `/health/ready` que ejecute `db.PingContext` con timeout y devuelva 503 si falla.

### 5.1–5.5 — Frontend no es desplegable desde la rama entregable

- **Qué falta:** frontend versionado, Dockerfile multi-stage, configuración de API y servicio Compose.
- **Por qué es crítico:** una aplicación web sólo está desplegada como sistema si interfaz y API se construyen/arrancan en sus contenedores previstos.
- **Cómo corregirlo:** agregue el servicio `frontend` y valide su imagen en CI y Compose antes de promover a `main`.

## ⚠️ HALLAZGOS MENORES (no bloqueantes)

- **1.4:** normalizar el commit inicial y documentar/validar Conventional Commits (por ejemplo con commitlint). Los últimos commits ya muestran una dirección correcta: `docs:` y `chore:`.
- **1.7:** el barrido de patrones de alta confianza no encontró tokens, pero falta evidencia fuerte: añada `gitleaks` a CI y, de ser posible, a un hook pre-commit. Esto convierte una revisión puntual en un control continuo.

## ✅ PUNTOS FUERTES

- **1.1:** hay repositorio Git operativo e historial consultable.
- **1.3:** el equipo ya usa ramas de funcionalidad con nombres claros (`feat/s1-...`), una buena base para formalizar Git Flow.
- Los commits de planificación más recientes ya emplean prefijos semánticos, aunque aún no de forma completa.

## Evidencia adicional: worktree no fusionado `codex/s1-integration`

Esta sección **no modifica el puntaje oficial**. Sirve para evitar perder trabajo ya existente y para explicar qué debe corregirse antes de fusionarlo a `main`.

### Controles presentes en el candidato

- Incluye `.github/workflows/ci.yml`, `.env.example`, `compose.yaml`, `compose.production.yaml`, `backend/Dockerfile`, `frontend/Dockerfile` y migraciones Flyway `V1__...` y `V2__...`.
- Compose declara PostgreSQL 18.6-alpine, volumen nombrado `postgres-data`, `pg_isready`, Flyway que espera `postgres: service_healthy`, y backend que espera `flyway: service_completed_successfully`.
- El backend usa multi-stage build, base final distroless no root, `USER nonroot:nonroot`, `EXPOSE 8080`, configuración por variables `TF_*` y logs mediante el paquete estándar de Go.
- `docker compose --profile app config --quiet` terminó correctamente y `go test ./...` del backend pasó en el host local.

### Bloqueos del candidato antes de fusionar

1. **El backend no construye en Docker.** Se ejecutó `docker build -t tallerflow-audit-backend:local backend` y falló en `go mod download`: `go.mod requires go >= 1.27.1 (running go 1.25.14; GOTOOLCHAIN=local)`. Corrija `backend/Dockerfile` para usar la versión declarada por `go.mod` (por ejemplo `FROM golang:1.27.1-alpine AS build`) o reduzca coherentemente la directiva `go` y dependencias. Después, ejecute el build en CI.
2. **No existe `.dockerignore`.** Docker envió contexto sin reglas de exclusión. Cree `backend/.dockerignore` y `frontend/.dockerignore` con al menos `.git`, `.env`, `node_modules`, `dist`, `coverage` y archivos de log.
3. **El frontend del worktree sólo contiene el Dockerfile.** No hay `frontend/package.json`, `package-lock.json` ni directorio `dist`; por tanto, su instrucción `COPY package.json package-lock.json ./` fallará. Integre la rama del frontend (o sus fuentes), fije versiones y pruebe su imagen.
4. **Readiness de BD no está conectada.** `cmd/api/main.go` crea el router con `httpx.Dependencies{}`; por ello `/health/ready` siempre devuelve 503 aunque exista `/health/live`. Inicialice el pool PostgreSQL y entregue `Ping: db.PingContext` al router. Mantenga `/health/live` para liveness y use `/health/ready` como criterio de disponibilidad.
5. **Secretos locales por defecto en Compose.** Valores como `local-*-change-me` son claramente de desarrollo, pero están hardcodeados como defaults. Para cumplir estrictamente el factor III y el ítem 4.3, elimine defaults de contraseña en `compose.yaml`, exija las variables (`${VAR:?mensaje}`) y deje valores ficticios sólo en `.env.example`.

## 📝 DICTAMEN FINAL

**¿Puede desplegarse en contenedores hoy? — No.**

- **Etapa alcanzada:** 🔴 **Nivel 0**, 2/45, para la única rama entregable (`main`).
- **Motivo:** `main` no contiene Dockerfile, Compose, código de aplicación, migraciones, pipeline o configuración de entornos. No hay artefacto que Docker pueda construir.
- **Observación decisiva:** incluso el candidato no fusionado falla actualmente al construir su backend por la incompatibilidad entre Go 1.25 de la imagen y Go 1.27.1 requerido por `go.mod`.

### Próximos 3 pasos concretos para subir al siguiente nivel

1. **Integrar de forma controlada:** cree `develop`, fusione por PR el código completo y los archivos de infraestructura del worktree de integración; no promocione a `main` hasta que CI esté verde.
2. **Hacer construible el sistema:** alinee la versión de Go del Dockerfile con `go.mod`, agregue ambos `.dockerignore`, integre los archivos reales del frontend y haga que CI construya backend y frontend.
3. **Validar el stack completo:** elimine secretos por defecto de Compose, conecte `/health/ready` a PostgreSQL y ejecute `docker compose --profile app up --build --wait`, una prueba HTTP y una prueba de persistencia tras `down`/`up`; etiquete entonces la entrega como `v0.1.0`.
