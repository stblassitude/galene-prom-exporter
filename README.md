# galene-prom-exporter

A Prometheus exporter for the [Galene](https://github.com/jech/galene)
videoconferencing server.

The exporter is a simple proxy: each scrape of `/metrics` queries Galene's
`/galene-api/v0/.stats` endpoint and translates the result into Prometheus
metrics. Nothing is cached or polled in the background.

## Building

    go build

## Running

The stats endpoint requires administrator credentials, either a user with
`"permissions": "admin"` in Galene's `data/config.json`, or an admin token.

    GALENE_USERNAME=admin GALENE_PASSWORD=secret \
        ./galene-prom-exporter -galene-url https://localhost:8443 -insecure

Metrics are then available at `http://localhost:9814/metrics`.

| Flag                | Environment              | Default                  | Description                                            |
|---------------------|--------------------------|--------------------------|--------------------------------------------------------|
| `-listen`           | `GALENE_EXPORTER_LISTEN` | `:9814`                  | Address to serve metrics on                            |
| `-metrics-path`     |                          | `/metrics`               | HTTP path for metrics                                  |
| `-galene-url`       | `GALENE_URL`             | `https://localhost:8443` | Base URL of the Galene server                          |
| `-username`         | `GALENE_USERNAME`        |                          | Admin user for HTTP Basic authentication               |
| `-password-file`    | `GALENE_PASSWORD`        |                          | File containing the admin password                     |
| `-token-file`       | `GALENE_TOKEN`           |                          | File containing an admin bearer token (takes priority) |
| `-insecure`         |                          | `false`                  | Skip TLS certificate verification (self-signed certs)  |
| `-timeout`          |                          | `5s`                     | Timeout for requests to Galene                         |
| `-track-metrics`    |                          | `true`                   | Export per-connection and per-track metrics            |

## Metrics

| Metric                                           | Labels                                           |
|--------------------------------------------------|--------------------------------------------------|
| `galene_up`                                      |                                                  |
| `galene_scrape_duration_seconds`                 |                                                  |
| `galene_groups`                                  |                                                  |
| `galene_group_clients`                           | `group`                                          |
| `galene_group_connections`                       | `group`, `direction`                             |
| `galene_group_tracks`                            | `group`, `direction`                             |
| `galene_group_bitrate_bits_per_second`           | `group`, `direction`                             |
| `galene_connection_max_bitrate_bits_per_second`  | `group`, `client`, `direction`, `connection`     |
| `galene_track_bitrate_bits_per_second`           | `group`, `client`, `direction`, `connection`, `track` |
| `galene_track_max_bitrate_bits_per_second`       | ditto                                            |
| `galene_track_loss_ratio`                        | ditto                                            |
| `galene_track_rtt_seconds` (down only)           | ditto                                            |
| `galene_track_jitter_seconds`                    | ditto                                            |
| `galene_track_{spatial,temporal}_layer`          | ditto (down only)                                |
| `galene_track_max_{spatial,temporal}_layer`      | ditto (down only)                                |

`direction` is `up` (client to server) or `down` (server to client).

Client and connection IDs are random and change with every session. With
many short sessions, the per-track metrics can create a lot of series; use
`-track-metrics=false` to export only group-level aggregates.

Galene documents the format of the stats endpoint as unstable, so a new Galene
version may need changes here.

## Debian package

Releases include `.deb` packages for amd64 and arm64. The package installs
a systemd unit, `galene-prom-exporter.service`, and the configuration file
`/etc/default/galene-prom-exporter`. Put the Galene URL and credentials
there, then run

    systemctl start galene-prom-exporter

The service is enabled on install but not started, since it needs
credentials first.
