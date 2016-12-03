package main

import (
	"encoding/json"
	"net"
	"strings"

	"github.com/eawsy/aws-lambda-go/service/lambda/runtime"
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

func handle(evt json.RawMessage, ctx *runtime.Context) (interface{}, error) {
	var ie IncomingEvent
	if err := json.Unmarshal(evt, &ie); err != nil {
		return nil, err
	}
	l, err := getLocation(ie.IP)
	output := strings.Join([]string{ie.IP, l.City.Names["en"], l.FullISO()}, ",")
	return output, err
}

func getLocation(ip string) (l Location, err error) {
	db, err := maxminddb.Open("GeoLite2-City.mmdb")
	if err != nil {
		return
	}
	defer db.Close()
	parsedIp := net.ParseIP(ip)
	err = db.Lookup(parsedIp, &l)
	if err != nil {
		return
	}
	return
}

func init() {
	runtime.HandleFunc(handle)
}

func main() {}
