# TallerFlow MVP — Hoja de ruta de implementación

**Estado:** Activa

**Especificación:** `docs/superpowers/specs/2026-09-21-tallerflow-mvp-design.md`

**Objetivo:** Entregar TallerFlow como un piloto seguro, recuperable y desplegable sin depender de un Backend-as-a-Service.

Esta hoja de ruta ordena el MVP, pero no sustituye los planes ejecutables de cada fase. Cada fase tendrá su propio plan detallado, pruebas, rama de integración y criterio de salida.

## Principios de ejecución

- Cada fase produce software integrado y demostrable.
- Flyway es la única autoridad del esquema.
- Las reglas se implementan primero con pruebas.
- Frontend y backend comparten OpenAPI, no tipos copiados manualmente.
- Las consultas privadas se aíslan por aplicación y PostgreSQL RLS.
- Ninguna fase introduce servicios previstos para fases posteriores.
- Toda rama parte de `main` actualizado y entra mediante pull request.
- Un entregable no se considera terminado mientras solo exista en una rama personal.

## Fase 1 — Cimientos seguros

**Plan:** `docs/superpowers/plans/2026-09-21-tallerflow-phase-1-foundations.md`

**Incluye:**

- PostgreSQL, Flyway y roles separados.
- Estructura modular de Go.
- Health checks, errores y request IDs.
- Usuarios, credenciales, sesiones, CSRF, talleres y membresías.
- RLS y contexto transaccional del taller.
- Bootstrap del primer OWNER.
- Vue 3 + TypeScript, sistema visual, login y sesión.
- Docker Compose, Caddy local y CI.

**Salida demostrable:** un OWNER creado por bootstrap inicia sesión, consulta su taller y cierra sesión; una petición sin sesión, sin CSRF o de otro taller es rechazada.

## Fase 2 — Operación básica

**Plan futuro:** `docs/superpowers/plans/2026-09-21-tallerflow-phase-2-orders.md`

**Incluye:**

- Clientes.
- Órdenes y código secuencial.
- Plantilla de siete etapas.
- Listados, filtros y paginación.
- Dashboard inicial.
- Crear orden desde escritorio y móvil.

**Salida demostrable:** OWNER o ADMIN crea un cliente y una orden completa; OPERATOR no puede hacerlo y otro taller recibe `404`.

## Fase 3 — Producción trazable

**Plan futuro:** `docs/superpowers/plans/2026-09-21-tallerflow-phase-3-production.md`

**Incluye:**

- Asignación de responsables.
- Avances incrementales e idempotencia.
- Correcciones append-only.
- Problemas, pausa y reanudación.
- Cambio de etapa y finalización.
- Historial y auditoría.

**Salida demostrable:** un operario registra un avance desde móvil y el dueño ve el historial sin duplicados ni ediciones destructivas.

## Fase 4 — Cliente y evidencias

**Plan futuro:** `docs/superpowers/plans/2026-09-21-tallerflow-phase-4-tracking.md`

**Incluye:**

- Adaptador privado de archivos.
- Validación de imágenes.
- Visibilidad interna o pública.
- Token de tracking almacenado como hash.
- Proyección pública y regeneración de enlace.

**Salida demostrable:** el cliente abre un enlace privado y solo ve datos y evidencias autorizados.

## Fase 5 — Piloto productivo

**Plan futuro:** `docs/superpowers/plans/2026-09-21-tallerflow-phase-5-production.md`

**Incluye:**

- Dominio y DNS.
- Caddy productivo.
- VPS de aplicación y VPS de datos.
- WireGuard o red privada.
- pgBackRest y backup externo.
- Restauración, alertas, endurecimiento y E2E.
- Manual operativo y rollback.

**Salida demostrable:** el piloto corre con HTTPS, backup externo y restauración verificada.

## Dependencias

```text
Fase 1: identidad y plataforma
  └── Fase 2: clientes y órdenes
        └── Fase 3: producción
              └── Fase 4: tracking y archivos
                    └── Fase 5: piloto productivo
```

Las tareas de diseño visual pueden avanzar con fixtures, pero la integración respeta este orden.

## Control entre fases

Antes de abrir el plan siguiente:

1. Todas las pruebas de la fase actual pasan.
2. Docker levanta el recorrido demostrable desde un entorno limpio.
3. El contrato OpenAPI refleja lo desplegado.
4. No quedan defectos críticos o altos.
5. La documentación de operación está actualizada.
6. El equipo etiqueta el punto de salida.

## Distribución de alto nivel

| Persona | Responsabilidad sostenida |
|---|---|
| José | Plataforma Go, autenticación, seguridad HTTP e integración |
| Lucero | Vue, sistema visual, accesibilidad y pruebas frontend |
| Michael | Dominio Go, multi-tenancy, contratos y reglas de negocio |
| Stephano | PostgreSQL, Flyway, Docker, Caddy, CI y operación |

La distribución concreta y los límites de archivos se definen en cada Sprint para reducir conflictos.
