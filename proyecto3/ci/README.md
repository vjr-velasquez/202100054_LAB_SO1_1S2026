# Validación reutilizable

`sh proyecto3/ci/check.sh` ejecuta `make check`: referencia Python, compilación,
pruebas Go, go vet y pruebas Rust mediante Docker. Requiere Python >=3.10, Make,
Docker y acceso a las imágenes/dependencias durante la primera construcción.

`make -C proyecto3 check-reference` es la alternativa rápida sin Docker; su alcance
no incluye servicios. `make -C proyecto3 check-integration` levanta el stack local y
comprueba el flujo y sus fallos. No hay workflow CI remoto configurado todavía.
