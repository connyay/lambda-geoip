package main

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/oschwald/maxminddb-golang"
)

const (
	liteURL       = "http://geolite.maxmind.com/download/geoip/database/GeoLite2-City.mmdb.gz"
	commercialURL = "https://www.maxmind.com/app/geoip_download?edition_id=GeoIP2-City&suffix=tar.gz&license_key="
	geoLiteName   = "GeoLite2-City.mmdb"
)

var (
	target    string
	url       string
	cacheName string
)

func main() {
	args := os.Args
	if len(args) < 2 || (args[1] != "lite" && args[1] != "commercial") {
		log.Fatalln("Please specify target (lite or commercial)")
	}
	target = args[1]
	switch target {
	case "lite":
		url = liteURL
		cacheName = "geolite"
	case "commercial":
		license := os.Getenv("MAXMIND_LICENSE")
		if license == "" {
			log.Fatalln("MAXMIND_LICENSE is not set")
		}
		url = commercialURL + license
		cacheName = "geoip.tar"
	}

	p := filepath.Join(os.Getenv("HOME"), ".cache", cacheName+".gz")

	client := http.DefaultClient
	// Assume we need to download the file
	download := true

	gmt := time.FixedZone("GMT", 0)

	stat, err := os.Stat(p)
	if err == nil {
		modTime := stat.ModTime().In(gmt).Format(time.RFC1123)
		req, _ := http.NewRequest("HEAD", url, nil)
		req.Header.Add("If-Modified-Since", modTime)

		res, err := client.Do(req)
		if err != nil {
			log.Fatalf("Error checking tarball last modified: %s", err)
		}
		if res.StatusCode == 304 {
			// We have the latest tarball. No need to redownload
			log.Println("Already have latest tarball - skipping download")
			download = false
		} else {
			log.Println("We have tarball, but is not latest")
		}
		if res.StatusCode == 401 {
			log.Fatalf("Unauthorized HEAD request")
		}
	} else if os.IsNotExist(err) {
		log.Printf("No tarball found at %s", p)
	}

	// We either do not have the tarball, or the tarball is out of date
	if download {
		log.Println("Starting tarball download")
		file, err := os.Create(p)
		defer file.Close()
		if err != nil {
			log.Fatalf("Error opening tarball creation: %s", err)
		}
		res, err := http.Get(url)
		defer res.Body.Close()
		if res.StatusCode == 401 {
			log.Fatalf("Unauthorized GET request")
		}
		if err != nil {
			log.Fatalf("Error downloading tarball: %s", err)
		}
		_, err = io.Copy(file, res.Body)
		if err != nil {
			log.Fatalf("Error writing downloaded tarball: %s", err)
		}
		lastMod := res.Header.Get("Last-Modified")
		modTime, err := time.ParseInLocation(time.RFC1123, lastMod, gmt)
		if err == nil {
			log.Printf("Setting tarball last modified to: %s", lastMod)
			os.Chtimes(p, time.Now(), modTime)
		}
	}
	file, err := os.Open(p)
	defer file.Close()
	if err != nil {
		log.Fatalf("Error opening tarball for extraction: %s", err)
	}

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		log.Fatalf("Error opening tarball reader: %s", err)
	}
	defer gzReader.Close()
	if target == "commercial" {
		tarBallReader := tar.NewReader(gzReader)
	tarLoop:
		for {
			header, err := tarBallReader.Next()
			if err != nil {
				if err == io.EOF {
					break
				}
				log.Fatalf("Error reading tarball: %s", err)
			}
			filename := header.Name

			switch header.Typeflag {
			case tar.TypeReg:
				// We only care about the mmdb file
				if filepath.Ext(filename) == ".mmdb" {
					_, name := path.Split(filename)
					writeDB(name, tarBallReader)
					checkDB(name)
					break tarLoop
				}
			}
		}
	} else {
		writeDB(geoLiteName, gzReader)
		checkDB(geoLiteName)
	}
}

func writeDB(name string, r io.Reader) {
	log.Printf("Extracting to %s", name)
	f, err := os.Create(name)
	if err != nil {
		log.Fatalf("Error creating database: %s", err)
	}
	_, err = io.Copy(f, r)
	if err != nil {
		log.Fatalf("Error writing database: %s", err)
	}
	f.Close()
}

func checkDB(name string) {
	b, err := ioutil.ReadFile(name)
	if err != nil {
		log.Fatalf("Error reading MaxMind database: %s", err)
	}
	_, err = maxminddb.FromBytes(b)
	if err == nil {
		log.Println("Successfully initialized MaxMind database")
	} else {
		log.Fatalf("Error initializing MaxMind database: %s", err)
	}
}
