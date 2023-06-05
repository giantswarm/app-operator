# Directories.
SCRIPTS_DIR := hack

sync-chart-crd:
	@echo "$(GEN_COLOR)Sync Chart CRD with apiextensions-application$(NO_COLOR)"
	cd $(SCRIPTS_DIR); ./sync-chart-crd.sh
