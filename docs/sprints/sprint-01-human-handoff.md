# TallerFlow — Sprint 1: handoff humano para José

**Propósito:** dejar listo el entorno, los accesos y la coordinación humana necesarios para ejecutar el Sprint 1 sin bloquear al equipo ni exponer secretos.

**Estado técnico local (2026-09-26):** entorno y recorrido completo verificados; publicación y comprobación del CI remoto pendientes.

**Fuentes:** especificación MVP, plan ejecutable de Fase 1 y plan de equipo del Sprint 1.

## 1. Preparar la estación de trabajo

Ejecutar PowerShell en una terminal normal. Usar una terminal como administrador solo si el instalador lo solicita.

### Go 1.27.x

```powershell
winget source update
winget search --id GoLang.Go --exact
winget show --id GoLang.Go --exact --versions
```

Si `winget` no está disponible, usar el instalador oficial de [Go para Windows](https://go.dev/dl/) y elegir la revisión estable 1.27.x.

Elegir la revisión estable disponible de la serie `1.27` y reemplazar `<1.27.x>`:

```powershell
winget install --id GoLang.Go --exact --version <1.27.x>
```

Cerrar y volver a abrir PowerShell; después verificar:

```powershell
go version
go env GOPATH
```

El primer comando debe informar `go1.27.x`.

### Node.js 24 LTS

```powershell
winget search --id OpenJS.NodeJS.LTS --exact
winget show --id OpenJS.NodeJS.LTS --exact --versions
```

Si `winget` no está disponible, usar el instalador oficial de [Node.js 24 LTS](https://nodejs.org/en/download).

Elegir una revisión estable disponible de la serie `24` y reemplazar `<24.x.x>`:

```powershell
winget install --id OpenJS.NodeJS.LTS --exact --version <24.x.x>
```

Cerrar y volver a abrir PowerShell; después verificar:

```powershell
node --version
npm --version
```

`node --version` debe comenzar con `v24.`.

### Docker Desktop y Compose

Docker Desktop ya está instalado en esta máquina. Abrirlo desde el menú Inicio y esperar a que el motor esté activo. En otra máquina donde falte, usar la [instalación oficial para Windows](https://docs.docker.com/desktop/setup/install/windows-install/). Si solicita habilitar WSL 2 o reiniciar Windows, completar ese paso antes de continuar.

```powershell
wsl --status
docker version
docker compose version
docker info
```

`docker version` debe mostrar cliente y servidor, `docker compose version` debe funcionar y `docker info` debe terminar sin error de conexión. La versión instalada de Compose puede ser superior a v2. No hace falta comprar ni configurar VPS para este sprint.

### Repositorio local

```powershell
git --version
git status --short
git branch --show-current
```

Antes de coordinar integraciones, confirmar que cada persona trabaja en su rama y que no hay cambios ajenos mezclados. No limpiar, descartar ni sobrescribir cambios de otra persona.

## 2. Preparar secretos locales de forma segura

Los secretos reales no se escriben en comandos, historial, capturas, chats, issues, PR, commits ni archivos rastreados. Los nombres esperados en Fase 1 incluyen:

- `TF_DATABASE_URL`
- `TF_SESSION_PEPPER`
- `DB_OWNER_PASSWORD`
- `DB_MIGRATION_PASSWORD`
- `DB_APP_PASSWORD`
- `DB_BOOTSTRAP_PASSWORD`
- `E2E_OWNER_PASSWORD`
- `E2E_NEW_PASSWORD`

Para una ejecución local puntual, introducir cada valor de forma oculta y mantenerlo solo en el proceso actual de PowerShell:

```powershell
function Set-ProcessSecret {
    param([Parameter(Mandatory)][string]$Name)
    $secret = Read-Host "Valor para $Name" -AsSecureString
    $pointer = [Runtime.InteropServices.Marshal]::SecureStringToBSTR($secret)
    try {
        $plain = [Runtime.InteropServices.Marshal]::PtrToStringBSTR($pointer)
        [Environment]::SetEnvironmentVariable($Name, $plain, 'Process')
    }
    finally {
        if ($pointer -ne [IntPtr]::Zero) {
            [Runtime.InteropServices.Marshal]::ZeroFreeBSTR($pointer)
        }
        Remove-Variable plain, secret -ErrorAction SilentlyContinue
    }
}

Set-ProcessSecret TF_SESSION_PEPPER
Set-ProcessSecret DB_OWNER_PASSWORD
Set-ProcessSecret DB_MIGRATION_PASSWORD
Set-ProcessSecret DB_APP_PASSWORD
Set-ProcessSecret DB_BOOTSTRAP_PASSWORD
Set-ProcessSecret E2E_OWNER_PASSWORD
Set-ProcessSecret E2E_NEW_PASSWORD
```

Comprobar solo la presencia, nunca imprimir el contenido:

```powershell
$required = 'TF_SESSION_PEPPER', 'DB_OWNER_PASSWORD', 'DB_MIGRATION_PASSWORD', 'DB_APP_PASSWORD', 'DB_BOOTSTRAP_PASSWORD'
$required | ForEach-Object {
    [pscustomobject]@{
        Variable = $_
        Configurada = -not [string]::IsNullOrWhiteSpace([Environment]::GetEnvironmentVariable($_, 'Process'))
    }
}
```

Las variables desaparecen al cerrar esa terminal. Si el equipo decide usar un archivo local, esperar a que la rama de Stefano defina `.env.example` y las reglas de exclusión; antes de guardar valores, exigir que `git check-ignore <archivo-local>` devuelva la ruta. Si no la devuelve, no crear ni rellenar el archivo.

Antes de cada commit o PR:

```powershell
git status --short
git diff --cached --check
git diff --cached --name-only
```

Revisar manualmente que la lista preparada no contenga archivos locales, volcados, claves, certificados ni credenciales. Nunca ejecutar comandos que impriman todas las variables de entorno durante una demo o revisión.

## 3. Confirmar accesos

José debe verificar:

- Acceso de lectura y escritura al repositorio remoto para Lucero, Michael y Stefano.
- Permiso de cada integrante para crear su rama y abrir o actualizar PR.
- CI visible para todo el equipo y permisos para consultar logs sin datos sensibles.
- Reglas de protección de `main`: cambios por PR, revisión requerida y CI verde antes de fusionar.
- Un canal de coordinación para bloqueos, cambios de contrato y orden de merge; allí se comparten estados y referencias, no credenciales.
- Revisor asignado desde el inicio de cada PR, no al final del sprint.

Comprobaciones locales sin modificar el remoto:

```powershell
git remote -v
git fetch --all --prune
git branch --all
```

Si falta un acceso, resolverlo con el administrador del repositorio antes de que la persona empiece su entregable.

## 4. Coordinar ramas y PR

Las ramas acordadas son:

| Persona | Rama | Entrega principal |
|---|---|---|
| José | `feat/s1-jose-auth-api` | plataforma Go, autenticación y bootstrap |
| Lucero | `feat/s1-lucero-frontend-shell` | shell Vue, diseño, sesión y login |
| Michael | `feat/s1-michael-tenancy-team` | conexión, tenant runner, taller y roles |
| Stefano | `feat/s1-stefano-data-platform` | PostgreSQL, Flyway, Compose, Caddy y CI |

Cada persona crea su rama desde el mismo `main` actualizado:

```powershell
git switch main
git pull --ff-only
git switch -c feat/s1-<persona>-<area>
```

Acuerdos que José debe hacer visibles en las descripciones de PR:

- Stefano controla la numeración Flyway y revisa columnas o variables nuevas.
- José controla el router central y la composición de dependencias.
- Michael implementa el tenant runner; los módulos exponen `RegisterRoutes` y no cablean rutas globales.
- Lucero consume contratos coordinados; cualquier cambio de DTO se refleja primero en OpenAPI.
- Ninguna rama incorpora clientes, órdenes, avances, tracking, imágenes o despliegue en VPS durante Sprint 1.

Revisión principal:

| Autor | Revisor principal | Foco |
|---|---|---|
| José | Michael | seguridad, interfaces y testabilidad |
| Lucero | José | contrato HTTP, errores, sesión y accesibilidad |
| Michael | José | RLS, principal, roles y transacciones |
| Stefano | Michael | SQL, grants, Compose y reproducibilidad |

Orden de integración:

1. Stefano: PostgreSQL, Flyway y esquema.
2. José: backend base, criptografía y sesiones.
3. Michael: tenancy, principal y roles.
4. Lucero: frontend y login.
5. Stefano: Compose, Caddy y CI integrados.
6. José: integración y E2E.

Después de fusionar las cuatro ramas personales en `main`, José crea la rama final desde un `main` nuevamente actualizado:

```powershell
git switch main
git pull --ff-only
git switch -c codex/s1-sprint1-final
```

No crear esa rama desde la rama personal de José ni antes de integrar las cuatro entregas.

## 5. Revisiones y puntos de control

### Días 1–2

- Go y Vue compilan.
- Flyway migra una base vacía.
- Las interfaces de autenticación y tenancy quedan acordadas.
- Las versiones instaladas y Docker quedan verificados.

### Días 3–5

- El token de sesión sin hash no aparece en PostgreSQL.
- RLS rechaza ausencia de contexto.
- El frontend no usa `localStorage` para sesión o credenciales.
- El login inválido usa un mensaje genérico.

### Días 6–8

- OWNER inicia sesión y `/me` devuelve taller y rol.
- Logout revoca la sesión.
- Una mutación sin CSRF devuelve `403`.
- Compose, Caddy y CI están listos para integración.

### Días 9–10

- Se respeta el orden de merge.
- La rama de integración nace del `main` actualizado.
- E2E incluye login, cambio obligatorio, logout, credenciales inválidas, sesión expirada y 360 px.
- El entorno levanta desde volúmenes vacíos y los runbooks reflejan los comandos comprobados.

## 6. Decisiones diferidas

Durante Sprint 1 no se compra, contrata ni configura dominio, DNS, VPS ni destino de backups. La selección del proveedor de dominio, proveedor de VPS y almacenamiento externo se decide antes de la **Fase 5**, comparando región, soporte, snapshots, red privada y costo.

La arquitectura objetivo conserva `app.tallerflow.pe`, Caddy, un VPS de aplicación, un VPS de datos y backup externo cifrado, pero los proveedores concretos y sus accesos quedan fuera de este handoff y del Sprint 1.

## 7. Criterios de listo para José

- [x] `go version` informa Go 1.27.x.
- [x] `node --version` informa Node 24.x LTS y `npm --version` funciona.
- [x] Docker Desktop está iniciado; Docker y Compose responden y ejecutan el stack.
- [x] La verificación usa credenciales descartables y no imprime las contraseñas de bootstrap.
- [ ] Los cuatro integrantes confirman acceso de escritura, visibilidad del CI y revisión correspondiente en GitHub.
- [x] Las cuatro ramas personales fueron obtenidas y sus entregables quedaron reconciliados.
- [ ] Los PR remotos declaran pruebas ejecutadas, cambios de contrato y dependencias.
- [x] La integración se realizó en `codex/s1-sprint1-final` sin modificar el `main` local durante la revisión.
- [x] Dominio, VPS y proveedores de backup permanecen diferidos a Fase 5.
- [x] La aceptación local pasó con Compose vacío, Flyway, bootstrap, login, cambio de contraseña, logout, CSRF, rate limit, RLS, accesibilidad y E2E.
- [ ] El workflow de CI remoto pasa sobre el commit publicado.
