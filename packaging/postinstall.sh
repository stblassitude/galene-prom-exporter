#!/bin/sh
set -e

if [ -d /run/systemd/system ]; then
	systemctl daemon-reload
	if [ "$1" = "configure" ] && [ -z "$2" ]; then
		# Fresh install: enable, but leave starting to the admin, since
		# credentials need to be configured first.
		systemctl enable galene-prom-exporter.service
	elif systemctl is-active --quiet galene-prom-exporter.service; then
		systemctl restart galene-prom-exporter.service
	fi
fi
