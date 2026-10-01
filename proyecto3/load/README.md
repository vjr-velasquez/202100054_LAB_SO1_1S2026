# Locust

`locustfile.py` genera los cinco países y los rangos de `load-profile.json`.
Cada petición lleva UUID y un identificador de corrida. Solo se considera correcta
una respuesta 202 con la identidad esperada.

Con el stack local activo, desde la raíz del repositorio:

```bash
RUN_ID=locust-prueba docker compose -f proyecto3/deploy/compose.local.yaml run --rm locust
```

El perfil Compose ejecuta dos usuarios durante diez segundos. Es una comprobación
básica, no la evaluación de rendimiento o HPA de la fase 6. El archivo de perfil se
lee localmente; su publicación/descarga como OCI corresponde a la fase 5.
