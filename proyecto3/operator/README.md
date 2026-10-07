# Operator Helm — fase 4

Operator real generado con Operator SDK v1.42.3 (plugin Helm), grupo
`monitoring.so1.local/v1alpha1`. Cada kind reconcilia su propio chart:

- `ValkeyInstance/store`: Service y StatefulSet `store-valkey`, AOF y PVC de 1 GiB.
- `GrafanaInstance/view`: Service y Deployment `view-grafana`, datasource Infinity,
  dashboard declarativo y adaptador Go que consulta Valkey.

El scaffold se creó con `operator-sdk init --plugins helm --domain so1.local
--project-name mumn-operator` y dos llamadas a `operator-sdk create api --group
monitoring --version v1alpha1 --kind ValkeyInstance` / `GrafanaInstance`.
Los charts y esquemas del scaffold se reemplazaron por los recursos del proyecto.
Referencia: [Operator SDK Helm](https://sdk.operatorframework.io/docs/building-operators/helm/quickstart/).

## Ejecutar y verificar

Desde la raíz del repositorio:

```bash
make -C proyecto3 phase4-up
make -C proyecto3 check-charts
make -C proyecto3 check-cluster
```

Requiere Docker, Python y Make; descarga herramientas con checksums oficiales en
`artifacts/tools`. Crea exclusivamente el clúster kind `mumn-phase4`, Kubernetes
1.35.0, con kubeconfig local `artifacts/kubeconfig`. Las imágenes locales se
importan para linux/amd64. Este entorno no es un despliegue en GKE.

`config/default` instala los CRD y `config/local`: Operator en namespace `mumn`
con RBAC limitado al namespace. Los demás directorios del scaffold SDK se
conservan como referencia y no forman parte de esa instalación.
Los ejemplos están en `config/samples`. `deploy/phase4/validation.yaml` instala
RabbitMQ de prueba, consumidor real y un Job que publica cinco reportes de fixture.
El publicador de prueba no sustituye la entrada Rust/Go verificada en fase 3.

La aceptación comprueba consultas reales de los 11 paneles a Infinity, cambio de
CR, recreación de Service, reinicio de Valkey con UID nuevo y eliminación/recreación
de ambos CR conservando el mismo PVC y sus cinco eventos.
Resultados locales: `artifacts/phase4-results.json`.

## Abrir Grafana

```bash
proyecto3/artifacts/tools/kubectl --kubeconfig proyecto3/artifacts/kubeconfig -n mumn port-forward service/view-grafana 13300:3000
```

Abrir http://localhost:13300/d/mumn-202100054 y entrar como `admin`.
Consultar la contraseña generada localmente en otra terminal:

```bash
proyecto3/artifacts/tools/kubectl --kubeconfig proyecto3/artifacts/kubeconfig -n mumn get secret grafana-admin -o jsonpath='{.data.password}' | base64 -d
```

Seleccionar corrida `fixture` y un rango que incluya 29/09/2026, 12:00–12:04 UTC.
Los paneles de resumen representan toda la corrida; solo la serie temporal
filtra por el rango del dashboard. Se muestran todas las modas empatadas y su
frecuencia. Los rankings usan el último reporte por país, con desempate temporal
por UUID. Los ceros son reportes válidos; una corrida vacía muestra ausencia de datos.

## Persistencia y límites

El PVC Valkey se retiene al eliminar el CR o reducir réplicas. Recrear `store` en
el mismo namespace reutiliza el PVC; eliminar el clúster kind elimina los datos
locales. AOF cada segundo no garantiza pérdida cero frente a caída abrupta.
Una instancia no ofrece alta disponibilidad. `storageSize` configura la creación;
el StatefulSet no admite cambiar su plantilla de PVC directamente.

Grafana tiene límite de 1 GiB: 512 MiB produjo OOM al abrir el dashboard durante
la prueba visual local. Debe ajustarse según mediciones de carga.
Grafana no tiene PVC: los dashboards y datasource se reconstruyen desde ConfigMaps;
los cambios manuales y sesiones no se conservan al recrear el pod. Las credenciales
permanecen en el Secret. El adaptador calcula estadísticas leyendo los eventos de
una corrida; su costo crece con el número de reportes y debe medirse en fase de carga.
RabbitMQ de validación es efímero y sus credenciales son solo para este clúster local.

Quedan para fase 5 GCP/GKE, publicación en Zot, Gateway API y despliegue del flujo
completo. HPA, carga y topología distribuida corresponden a las fases posteriores.
