package main

import (
	"net"
	"strings"

	"github.com/eawsy/aws-lambda-go-core/service/lambda/runtime"
	"github.com/oschwald/maxminddb-golang"
)

// Location captures the relevant data from the GeoIP database lookup.
type Location struct {
	City struct {
		Names map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		ISO string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	Subdivisions []struct {
		ISO string `maxminddb:"iso_code"`
	} `maxminddb:"subdivisions"`
}

// FullISO returns the full ISO-3166 code from the
// location i.e. GB-WLS
func (l *Location) FullISO() string {
	if len(l.Subdivisions) == 0 {
		// Empty location returned from MaxMind.
		return l.Country.ISO
	}

	return (l.Country.ISO + "-" + l.Subdivisions[0].ISO)
}

type IncomingEvent struct {
	IP string `json:"source-ip"`
}

var db *maxminddb.Reader

func init() {
	db, _ = maxminddb.Open("GeoLite2-City.mmdb")
}

func Handle(evt *IncomingEvent, ctx *runtime.Context) (interface{}, error) {
	l, err := getLocation(evt.IP)
	output := strings.Join([]string{evt.IP, l.City.Names["en"], l.FullISO()}, ",")
	return output, err
}

func getLocation(ip string) (l Location, err error) {
	if db == nil {
		return
	}
	parsedIp := net.ParseIP(ip)
	err = db.Lookup(parsedIp, &l)
	if err != nil {
		return
	}
	return
}
