# Sprint 5: operación y métricas

## Uso en el taller

El propietario y el administrador pueden configurar entre 1 y 20 etapas, con
una explicación breve para el cliente. La plantilla se copia al crear una orden;
modificarla no cambia el recorrido de órdenes anteriores. El ejemplo de confección
incluye muestra, corte, confección, planchado, embolsado y entrega.

Cada orden admite un responsable del equipo y un motivo de bloqueo. Mientras
está bloqueada, no se completa una etapa. Las modificaciones de especificaciones
requieren un motivo y quedan en el historial, junto con el actor y la fecha.
Los cambios usan la versión de la orden para evitar sobrescribir otra edición.

El panel distingue pedidos y cantidades por unidad. Un pedido de 600 prendas
cuenta como una orden y 600 unidades; no se suman prendas, kilos, metros y docenas
en una cifra común. Las colas permiten consultar bloqueos, órdenes sin responsable
y carga por etapa. Las entregas del día usan la zona America/Lima.

Clientes conserva alineación en escritorio y una lectura por tarjeta en celular.
El buscador mantiene un único indicador de foco. Los CSV de órdenes y clientes
se limitan a los roles autorizados y protegen los campos frente a fórmulas.

## Seguimiento del cliente

Copiar el enlace recupera el mismo vínculo de la orden. Reemplazarlo o revocarlo
son acciones explícitas del propietario o administrador. Los enlaces antiguos
siguen funcionando durante la transición, salvo reemplazo, revocación o vencimiento.
Los enlaces vencen 30 días después de completar la orden; editar notas no amplía
ese plazo. El seguimiento explica el proceso, presenta el avance y confirma la
última consulta. No expone responsables ni motivos de bloqueo internos.

La página pública de eliminación se conserva en `/eliminar-cuenta`. El acceso
interno está en **Mi cuenta → Cuenta y privacidad**; se elimina el pie global que
aparecía bajo todas las pantallas.

## Métricas de plataforma

La consola combina fuentes distintas, identificadas en pantalla:

| Fuente | Qué permite observar |
| --- | --- |
| Base operativa | Talleres, altas, estado y órdenes persistidas. |
| Eventos de uso | Primeras aperturas, registros verificados, inicios de sesión e interacción con sesión. |
| CSV oficial de Google Play | Instalaciones y desinstalaciones informadas por Google. |

Una primera apertura no equivale a una descarga ni identifica a una persona única.
Los eventos se deduplican; el usuario puede desactivarlos en Cuenta y privacidad.
El cálculo de usuarios y talleres únicos se realiza sobre todo el período, sin
sumar los únicos de cada día. Las fechas de esta consola son UTC y se indican como
tales. Los datos sin cobertura o sin reporte se muestran como **Sin datos**.

Los talleres demo se clasifican explícitamente mediante una acción auditada. Su
actividad autenticada puede excluirse; las aperturas anónimas y los agregados Play
no permiten esa separación. Las fotos R2 se cuentan como registros; su tamaño no
se presenta como cero cuando no está disponible.

La importación automática de Google Play requiere el bucket de reportes y una
identidad autorizada con permisos de lectura. La importación manual admite el
CSV oficial de instalaciones, total o por código de versión, en UTF-8 o UTF-16.
El detalle técnico y los límites se documentan en
[`backend/internal/analytics/README.md`](../../backend/internal/analytics/README.md).

## Condiciones de publicación

La versión Android prevista es **1.1.0, código 3**. Los recursos incluidos en la
app requieren un AAB nuevo; desplegar la web no actualiza las instalaciones
Android existentes. La compilación conserva la identidad de firma del proyecto.

Antes de publicar el AAB, se debe contrastar la declaración de Seguridad de los
datos con la telemetría incluida: interacciones en la app, identificador aleatorio
de instalación y asociación al ID de usuario autenticado. El identificador no es
el ID publicitario de Android. La declaración debe reflejar las finalidades y
la elección del usuario realmente implementadas. También se deben revisar los
avisos específicos del AAB en Play Console; esta documentación no acredita que
Google los haya aprobado.

El despliegue continúa usando el entorno protegido `production`. No se modifican
los aprobadores de GitHub. La integración en `main` y el despliegue se realizan
después de superar las comprobaciones locales y de CI.

## Referencias de implementación

- [Contrato de operaciones y seguimiento](../contracts/openapi/sprint5/operations.yaml).
- [Contrato de telemetría y consola](../contracts/openapi/sprint5/analytics.yaml).
- [Seguridad de datos de Google Play](https://support.google.com/googleplay/android-developer/answer/10787469).
- [Informes oficiales de instalaciones](https://support.google.com/googleplay/android-developer/answer/6135870).
- [Prevención de fórmulas en CSV](https://community.owasp.org/attacks/CSV_Injection).
