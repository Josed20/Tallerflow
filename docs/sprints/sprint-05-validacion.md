# Sprint 5: validación local

Fecha: 24 de noviembre de 2026 (Lima).
Rama: `feat/s5-stephano-operacion-metricas`.
Base remota comprobada: `1aa7f4ee2bb2319c36bc88bcfe4f18a29e796ffb`.

## Comprobaciones aprobadas

| Comprobación | Resultado observado |
| --- | --- |
| Frontend, Vitest | 42 archivos y 188 pruebas aprobadas. |
| Compilación web | `vue-tsc -b` y `vite build` aprobados. |
| Backend | `go test -p 1 ./...` aprobado; módulos analytics, consola y órdenes repetidos tras el último ajuste. |
| Análisis Go | `go vet ./...` aprobado. |
| PostgreSQL real | Integración de analytics, autenticación, consola, órdenes, recuperación y registro aprobada. |
| Migraciones | Instalación desde cero hasta V21 y actualización a V22 aprobadas. |
| Contrato de producción | Backfill histórico bajo RLS, aislamiento, posiciones hasta 20, conservación de snapshots y purga aprobados. |
| Contrato de primer propietario | Purga real del usuario y conservación del marcador de instalación reclamada aprobados; fixtures revertidos. |
| Seguimiento público, Chrome | 2 casos aprobados a 360 px. |
| Operaciones y recuperación, Chrome | 17 casos aprobados; el caso restante de desborde se corrigió y pasó en una ejecución focalizada. |
| Alta inicial real | Aprobada con servicios locales y un margen de 90 segundos para la lentitud del equipo; ejecución observada de 7,5 segundos. |
| Registro y correo reales | Alta, verificación en Mailpit y acceso al panel aprobados. |
| Recorrido Sprint 5 real | Ocho etapas, snapshots previos, bloqueo, enlace recuperable, privacidad del seguimiento y acceso a Cuenta aprobados. |
| Contratos OpenAPI | Los cuatro contratos verificados son válidos; seis avisos no bloqueantes de metadatos y recomendaciones. |
| Cambios Git | `git diff --check` aprobado; documentos de validación previos, diagramas y videos del usuario excluidos del cambio. |

Las pruebas de navegador se ejecutaron con perfiles aislados, sin usar la sesión
personal de Chrome. Las pruebas reales usaron exclusivamente PostgreSQL y Mailpit
locales; no se modificaron talleres de producción.

## Continuación después del reinicio

1. Verificar espacio libre en C: y el estado de Docker/WSL. Al terminar esta
   sesión seguía siendo insuficiente para realizar con margen las compilaciones
   grandes; se recomendó liberar unos 10 GB.
2. Completar la prueba de carrera de Go y la construcción de las imágenes finales.
3. Sincronizar los recursos Android con `VITE_API_BASE_URL=https://tallerflowpe.com`,
   compilar y validar Android 1.1.0, código 3, y generar el AAB firmado.
4. Revisar los avisos concretos de Play Console. El control del navegador falló
   durante esta sesión y los avisos no se consideran revisados.
5. Actualizar Seguridad de los datos para reflejar las métricas realmente incluidas
   y comprobar las demás correcciones solicitadas por Google antes de publicar.
6. Integrar mediante la revisión y CI del repositorio. Después, fusionar en `main`
   y ejecutar el despliegue final. El entorno `production` conserva la aprobación
   de José; no se cambiaron permisos.

No se hizo push, merge ni despliegue durante esta validación. Tampoco se publicó
un AAB nuevo en Google Play. La descarga automática de informes Play requiere
credenciales y permisos autorizados; la importación manual ya está implementada.

## Recuperación del entorno

La unidad C: llegó a tener aproximadamente 150 MB libres. Se limpiaron cachés
regenerables de npm, compilación Docker y un toolchain temporal de pruebas.
Los APK, documentos, capturas y videos del proyecto se conservaron.

El reinicio de WSL requirió cerrar Docker y reiniciar administrativamente el
proceso verificado `wslservice`. Se comprobó después `WslService: Running`, las
distribuciones detenidas y Docker Server 29.1.2 respondiendo. No se desregistraron
distribuciones, ni se restableció Docker de fábrica, ni se borraron datos de usuario.
