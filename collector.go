package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const namespace = "galene"

var (
	groupLabels = []string{"group"}
	dirLabels   = []string{"group", "direction"}
	connLabels  = []string{"group", "client", "direction", "connection"}
	trackLabels = []string{"group", "client", "direction", "connection", "track"}

	upDesc = prometheus.NewDesc(namespace+"_up",
		"Whether the last query of the Galene stats endpoint succeeded.",
		nil, nil)
	scrapeDurationDesc = prometheus.NewDesc(namespace+"_scrape_duration_seconds",
		"Time taken to query the Galene stats endpoint.",
		nil, nil)
	groupsDesc = prometheus.NewDesc(namespace+"_groups",
		"Number of currently active groups.",
		nil, nil)

	groupClientsDesc = prometheus.NewDesc(namespace+"_group_clients",
		"Number of clients connected to a group.",
		groupLabels, nil)
	groupConnectionsDesc = prometheus.NewDesc(namespace+"_group_connections",
		"Number of up (client to server) or down (server to client) connections in a group.",
		dirLabels, nil)
	groupTracksDesc = prometheus.NewDesc(namespace+"_group_tracks",
		"Number of up or down tracks in a group.",
		dirLabels, nil)
	groupBitrateDesc = prometheus.NewDesc(namespace+"_group_bitrate_bits_per_second",
		"Sum of the estimated bitrates of all up or down tracks in a group.",
		dirLabels, nil)

	connMaxBitrateDesc = prometheus.NewDesc(namespace+"_connection_max_bitrate_bits_per_second",
		"Maximum bitrate of a connection.",
		connLabels, nil)

	trackBitrateDesc = prometheus.NewDesc(namespace+"_track_bitrate_bits_per_second",
		"Estimated bitrate of a track.",
		trackLabels, nil)
	trackMaxBitrateDesc = prometheus.NewDesc(namespace+"_track_max_bitrate_bits_per_second",
		"Maximum bitrate of a track.",
		trackLabels, nil)
	trackLossDesc = prometheus.NewDesc(namespace+"_track_loss_ratio",
		"Packet loss ratio of a track (0 to 1).",
		trackLabels, nil)
	trackRttDesc = prometheus.NewDesc(namespace+"_track_rtt_seconds",
		"Round-trip time of a track (down tracks only).",
		trackLabels, nil)
	trackJitterDesc = prometheus.NewDesc(namespace+"_track_jitter_seconds",
		"Jitter of a track.",
		trackLabels, nil)
	trackSidDesc = prometheus.NewDesc(namespace+"_track_spatial_layer",
		"Currently selected spatial layer of a simulcast/SVC down track.",
		trackLabels, nil)
	trackMaxSidDesc = prometheus.NewDesc(namespace+"_track_max_spatial_layer",
		"Highest available spatial layer of a down track.",
		trackLabels, nil)
	trackTidDesc = prometheus.NewDesc(namespace+"_track_temporal_layer",
		"Currently selected temporal layer of a down track.",
		trackLabels, nil)
	trackMaxTidDesc = prometheus.NewDesc(namespace+"_track_max_temporal_layer",
		"Highest available temporal layer of a down track.",
		trackLabels, nil)
)

// galeneCollector queries Galene synchronously each time Prometheus
// scrapes the exporter; nothing is cached between scrapes.
type galeneCollector struct {
	url          string
	username     string
	password     string
	token        string
	client       *http.Client
	trackMetrics bool
}

func (c *galeneCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- upDesc
	ch <- scrapeDurationDesc
	ch <- groupsDesc
	ch <- groupClientsDesc
	ch <- groupConnectionsDesc
	ch <- groupTracksDesc
	ch <- groupBitrateDesc
	if c.trackMetrics {
		ch <- connMaxBitrateDesc
		ch <- trackBitrateDesc
		ch <- trackMaxBitrateDesc
		ch <- trackLossDesc
		ch <- trackRttDesc
		ch <- trackJitterDesc
		ch <- trackSidDesc
		ch <- trackMaxSidDesc
		ch <- trackTidDesc
		ch <- trackMaxTidDesc
	}
}

func (c *galeneCollector) Collect(ch chan<- prometheus.Metric) {
	start := time.Now()
	groups, err := c.fetch()
	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc,
		prometheus.GaugeValue, time.Since(start).Seconds())
	if err != nil {
		log.Printf("Querying Galene: %v", err)
		ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 0)
		return
	}
	ch <- prometheus.MustNewConstMetric(upDesc, prometheus.GaugeValue, 1)
	ch <- prometheus.MustNewConstMetric(groupsDesc,
		prometheus.GaugeValue, float64(len(groups)))

	for _, g := range groups {
		ch <- prometheus.MustNewConstMetric(groupClientsDesc,
			prometheus.GaugeValue, float64(len(g.Clients)), g.Name)

		var conns, tracks, bitrate [2]float64
		for _, cl := range g.Clients {
			for i, dir := range []struct {
				name  string
				conns []conn
			}{{"up", cl.Up}, {"down", cl.Down}} {
				conns[i] += float64(len(dir.conns))
				for _, cn := range dir.conns {
					tracks[i] += float64(len(cn.Tracks))
					for _, t := range cn.Tracks {
						bitrate[i] += float64(t.Bitrate)
					}
					if c.trackMetrics {
						c.collectConn(ch, g.Name, cl.Id, dir.name, &cn)
					}
				}
			}
		}
		for i, dir := range []string{"up", "down"} {
			ch <- prometheus.MustNewConstMetric(groupConnectionsDesc,
				prometheus.GaugeValue, conns[i], g.Name, dir)
			ch <- prometheus.MustNewConstMetric(groupTracksDesc,
				prometheus.GaugeValue, tracks[i], g.Name, dir)
			ch <- prometheus.MustNewConstMetric(groupBitrateDesc,
				prometheus.GaugeValue, bitrate[i], g.Name, dir)
		}
	}
}

func (c *galeneCollector) collectConn(ch chan<- prometheus.Metric, group, client, dir string, cn *conn) {
	if cn.MaxBitrate > 0 {
		ch <- prometheus.MustNewConstMetric(connMaxBitrateDesc,
			prometheus.GaugeValue, float64(cn.MaxBitrate),
			group, client, dir, cn.Id)
	}
	for i, t := range cn.Tracks {
		lv := []string{group, client, dir, cn.Id, strconv.Itoa(i)}
		gauge := func(d *prometheus.Desc, v float64) {
			ch <- prometheus.MustNewConstMetric(d,
				prometheus.GaugeValue, v, lv...)
		}
		layer := func(d *prometheus.Desc, v *uint8) {
			if v != nil {
				gauge(d, float64(*v))
			}
		}

		gauge(trackBitrateDesc, float64(t.Bitrate))
		if t.MaxBitrate > 0 {
			gauge(trackMaxBitrateDesc, float64(t.MaxBitrate))
		}
		gauge(trackLossDesc, t.Loss)
		if dir == "down" {
			gauge(trackRttDesc, t.Rtt/1000)
		}
		gauge(trackJitterDesc, t.Jitter/1000)
		layer(trackSidDesc, t.Sid)
		layer(trackMaxSidDesc, t.MaxSid)
		layer(trackTidDesc, t.Tid)
		layer(trackMaxTidDesc, t.MaxTid)
	}
}

func (c *galeneCollector) fetch() ([]groupStats, error) {
	req, err := http.NewRequest("GET", c.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	} else if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%v: %v", c.url, resp.Status)
	}

	var groups []groupStats
	err = json.NewDecoder(resp.Body).Decode(&groups)
	if err != nil {
		return nil, fmt.Errorf("decoding stats: %w", err)
	}
	return groups, nil
}
