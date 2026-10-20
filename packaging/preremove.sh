#!/bin/sh
set -e

if [ -d /run/systemd/system ] && [ "$1" = "remove" ]; then
	systemctl disable --now galene-prom-exporter.service || true
fi
