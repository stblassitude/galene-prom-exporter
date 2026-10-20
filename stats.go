package main

// Mirror of the JSON structures returned by Galene's /galene-api/v0/.stats
// endpoint (see github.com/jech/galene/stats/stats.go).  The format is
// officially undocumented and may change between Galene versions.

type groupStats struct {
	Name    string   `json:"name"`
	Clients []client `json:"clients"`
}

type client struct {
	Id   string `json:"id"`
	Up   []conn `json:"up"`
	Down []conn `json:"down"`
}

type conn struct {
	Id         string  `json:"id"`
	MaxBitrate uint64  `json:"maxBitrate"`
	Tracks     []track `json:"tracks"`
}

type track struct {
	Sid        *uint8  `json:"sid"`
	MaxSid     *uint8  `json:"maxSid"`
	Tid        *uint8  `json:"tid"`
	MaxTid     *uint8  `json:"maxTid"`
	Bitrate    uint64  `json:"bitrate"`    // bits per second
	MaxBitrate uint64  `json:"maxBitrate"` // bits per second
	Loss       float64 `json:"loss"`       // ratio, 0..1
	Rtt        float64 `json:"rtt"`        // milliseconds
	Jitter     float64 `json:"jitter"`     // milliseconds
}
