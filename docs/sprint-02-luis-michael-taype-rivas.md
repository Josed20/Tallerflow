# Sprint 2 — Luis Michael Taype Rivas

Responsable integral de la entrega: **Luis Michael Taype Rivas**.

Esta implementación consolida en una sola rama los cuatro frentes del diseño del 27 de septiembre de 2026: onboarding, recuperación/rutas, operación de clientes y órdenes, y administración del equipo. Las decisiones de seguridad, RLS, Flyway, cookies opacas, CSRF, idempotencia y respuestas anti-enumeración se preservan.

## Recorrido de demostración

1. Abrir una instalación vacía en `/onboarding` y crear taller y OWNER.
2. Crear un cliente en `/app/clients`.
3. Crear una orden en `/app/orders/new` y consultar su semáforo.
4. Generar una invitación en `/app/team` y consumirla en `/join`.
5. Solicitar recuperación en `/forgot-password` y consumirla en `/reset-password`.
6. Visitar una URL inexistente y verificar el estado 404.

La API consolidada se documenta en `docs/contracts/openapi.yaml`; los contratos autocontenidos están en `docs/contracts/openapi/sprint2/`.

## Estado del frente de Luis

Las tareas `L-01` a `L-12` están implementadas y verificadas. La recuperación usa tokens criptográficos de un solo uso, conserva únicamente SHA-256 en PostgreSQL, entrega el enlace mediante `PasswordResetDelivery` y revoca todas las sesiones al cambiar la contraseña. Mailpit cubre desarrollo y el adaptador SMTP configurable cubre producción.

La verificación incluye contrato OpenAPI, contrato SQL, pruebas unitarias, pruebas de interfaz, recorrido E2E con Mailpit y comprobación de HTTP 404 real. Ejecutar `scripts/verify-sprint2.ps1` en Windows o `scripts/verify-sprint2.sh` en Linux/macOS.
