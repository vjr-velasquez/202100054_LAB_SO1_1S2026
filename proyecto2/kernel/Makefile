obj-m += continfo_pr2_so1_202100054.o

KDIR ?= /lib/modules/$(shell uname -r)/build
MODULE_DIR := $(CURDIR)
MODULE_NAME := continfo_pr2_so1_202100054
MODULE_FILE := $(MODULE_DIR)/$(MODULE_NAME).ko
PROC_FILE := /proc/$(MODULE_NAME)

SIGN_FILE ?= $(KDIR)/scripts/sign-file
MOK_PRIVATE_KEY ?= /var/lib/shim-signed/mok/MOK.priv
MOK_PUBLIC_KEY ?= /var/lib/shim-signed/mok/MOK.der

.PHONY: all sign load unload status validate clean

all:
	$(MAKE) -C $(KDIR) M=$(MODULE_DIR) modules

sign:
	@test -f "$(MODULE_FILE)" || \
		(echo "ERROR: primero debe compilar el módulo"; exit 1)
	@test -x "$(SIGN_FILE)" || \
		(echo "ERROR: no se encontró sign-file"; exit 1)
	@test -f "$(MOK_PRIVATE_KEY)" || \
		(echo "ERROR: no se encontró la clave privada MOK"; exit 1)
	@test -f "$(MOK_PUBLIC_KEY)" || \
		(echo "ERROR: no se encontró el certificado MOK"; exit 1)
	sudo "$(SIGN_FILE)" sha256 \
		"$(MOK_PRIVATE_KEY)" \
		"$(MOK_PUBLIC_KEY)" \
		"$(MODULE_FILE)"
	@test "$$(modinfo -F sig_id "$(MODULE_FILE)")" = "PKCS#7"
	@echo "Módulo firmado correctamente"

load:
	@if lsmod | grep -q "^$(MODULE_NAME)[[:space:]]"; then \
		echo "ERROR: el módulo ya está cargado"; \
		exit 1; \
	fi
	$(MAKE) all
	$(MAKE) sign
	sudo insmod "$(MODULE_FILE)"
	@echo "Módulo cargado: $(PROC_FILE)"

unload:
	@if lsmod | grep -q "^$(MODULE_NAME)[[:space:]]"; then \
		sudo rmmod "$(MODULE_NAME)"; \
		echo "Módulo descargado correctamente"; \
	else \
		echo "El módulo no estaba cargado"; \
	fi

status:
	@if lsmod | grep -q "^$(MODULE_NAME)[[:space:]]"; then \
		echo "Módulo activo"; \
	else \
		echo "Módulo inactivo"; \
	fi
	@if test -e "$(PROC_FILE)"; then \
		echo "$(PROC_FILE) disponible"; \
	else \
		echo "$(PROC_FILE) no disponible"; \
	fi

validate:
	@test -r "$(PROC_FILE)" || \
		(echo "ERROR: $(PROC_FILE) no está disponible"; exit 1)
	@python3 -m json.tool "$(PROC_FILE)" >/dev/null
	@echo "JSON válido en $(PROC_FILE)"

clean:
	@if lsmod | grep -q "^$(MODULE_NAME)[[:space:]]"; then \
		echo "ERROR: descargue el módulo antes de limpiar"; \
		exit 1; \
	fi
	$(MAKE) -C $(KDIR) M=$(MODULE_DIR) clean