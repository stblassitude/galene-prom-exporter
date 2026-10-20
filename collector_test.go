package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

// Sample output of Galene's .stats endpoint.
const sampleStats = `[{"name":"test","clients":[
 {"id":"c1",
  "up":[{"id":"u1","tracks":[{"bitrate":800000,"maxBitrate":1000000,"loss":0.01,"jitter":2.5},
                             {"bitrate":32000,"loss":0}]}],
  "down":[{"id":"d1","maxBitrate":2000000,"tracks":[{"sid":0,"maxSid":2,"tid":1,"maxTid":2,
           "bitrate":500000,"maxBitrate":1500000,"loss":0.5,"rtt":40,"jitter":1}]}]},
 {"id":"c2"}]}]
`

func newTestCollector(t *testing.T, handler http.HandlerFunc) *galeneCollector {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return &galeneCollector{
		url:          srv.URL + "/galene-api/v0/.stats",
		username:     "admin",
		password:     "secret",
		client:       srv.Client(),
		trackMetrics: true,
	}
}

func TestCollect(t *testing.T) {
	c := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
		u, p, ok := r.BasicAuth()
		if !ok || u != "admin" || p != "secret" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if r.URL.Path != "/galene-api/v0/.stats" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(sampleStats))
	})

	expected := `
# HELP galene_up Whether the last query of the Galene stats endpoint succeeded.
# TYPE galene_up gauge
galene_up 1
# HELP galene_groups Number of currently active groups.
# TYPE galene_groups gauge
galene_groups 1
# HELP galene_group_clients Number of clients connected to a group.
# TYPE galene_group_clients gauge
galene_group_clients{group="test"} 2
# HELP galene_group_bitrate_bits_per_second Sum of the estimated bitrates of all up or down tracks in a group.
# TYPE galene_group_bitrate_bits_per_second gauge
galene_group_bitrate_bits_per_second{direction="down",group="test"} 500000
galene_group_bitrate_bits_per_second{direction="up",group="test"} 832000
# HELP galene_group_tracks Number of up or down tracks in a group.
# TYPE galene_group_tracks gauge
galene_group_tracks{direction="down",group="test"} 1
galene_group_tracks{direction="up",group="test"} 2
# HELP galene_track_loss_ratio Packet loss ratio of a track (0 to 1).
# TYPE galene_track_loss_ratio gauge
galene_track_loss_ratio{client="c1",connection="d1",direction="down",group="test",track="0"} 0.5
galene_track_loss_ratio{client="c1",connection="u1",direction="up",group="test",track="0"} 0.01
galene_track_loss_ratio{client="c1",connection="u1",direction="up",group="test",track="1"} 0
# HELP galene_track_rtt_seconds Round-trip time of a track (down tracks only).
# TYPE galene_track_rtt_seconds gauge
galene_track_rtt_seconds{client="c1",connection="d1",direction="down",group="test",track="0"} 0.04
# HELP galene_track_max_spatial_layer Highest available spatial layer of a down track.
# TYPE galene_track_max_spatial_layer gauge
galene_track_max_spatial_layer{client="c1",connection="d1",direction="down",group="test",track="0"} 2
# HELP galene_connection_max_bitrate_bits_per_second Maximum bitrate of a connection.
# TYPE galene_connection_max_bitrate_bits_per_second gauge
galene_connection_max_bitrate_bits_per_second{client="c1",connection="d1",direction="down",group="test"} 2e+06
`
	err := testutil.CollectAndCompare(c, strings.NewReader(expected),
		"galene_up", "galene_groups", "galene_group_clients",
		"galene_group_bitrate_bits_per_second", "galene_group_tracks",
		"galene_track_loss_ratio", "galene_track_rtt_seconds",
		"galene_track_max_spatial_layer",
		"galene_connection_max_bitrate_bits_per_second")
	if err != nil {
		t.Error(err)
	}

	// Ensure the collector passes the registry's consistency checks.
	reg := prometheus.NewPedanticRegistry()
	reg.MustRegister(c)
	if _, err := reg.Gather(); err != nil {
		t.Error(err)
	}
}

func TestCollectDown(t *testing.T) {
	c := newTestCollector(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	})
	expected := `
# HELP galene_up Whether the last query of the Galene stats endpoint succeeded.
# TYPE galene_up gauge
galene_up 0
`
	err := testutil.CollectAndCompare(c, strings.NewReader(expected), "galene_up")
	if err != nil {
		t.Error(err)
	}
}
