package main

import "fmt"
import "encoding/csv"
import "os"
import "time"
import "strconv"
import "github.com/umahmood/haversine"
import "net/url"
import "github.com/hashicorp/go-retryablehttp"
import "encoding/json"
import "io"
import "io/ioutil"
import "log"

func main() {

	records := toCsvRecord(os.Stdin, ';')
	rows := toRow(records)
	groups := toGroupedRow(rows)
	walks := toWalk(groups)
	urls := toUrl(walks, os.Args[1])
	makeMap(urls)
	select {}
}

func toCsvRecord(reader io.Reader, delimiter rune) <-chan []string {
	csvReader := csv.NewReader(reader)
	csvReader.Comma = delimiter
	/*
		We don't care about the exact number of fields. We only care that at least 3 exist for a given row.
		No need to overspecify.
	*/
	csvReader.FieldsPerRecord = -1

	out := make(chan []string)
	go func() {
		defer close(out)
		for {
			record, err := csvReader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
			out <- record
		}
	}()
	return out
}

type row struct {
	lat  float64
	lon  float64
	time time.Time
}

func toRow(in <-chan []string) <-chan row {
	out := make(chan row)
	go func() {
		defer close(out)
		for record := range in {
			if len(record) < 3 {
				err := fmt.Errorf("record must have at least 3 fields %v", record)
				fmt.Fprintln(os.Stderr, err)
				continue
			}

			parseErrors := make([]error, 0, 3)
			//layout was found using dateparse.ParseFormat from https://github.com/araddon/dateparse.
			layout := "2006-01-02 15:04:05 -0700"
			time, err := time.Parse(layout, record[0])
			if err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("could not parse time for record %v: %w", record, err))
			}

			lat, err := strconv.ParseFloat(record[1], 64)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("could not parse latitude for record %v: %w", record, err))
			}

			lon, err := strconv.ParseFloat(record[2], 64)
			if err != nil {
				parseErrors = append(parseErrors, fmt.Errorf("could not parse longitude for record %v: %w", record, err))
			}

			if len(parseErrors) != 0 {
				for _, e := range parseErrors {
					fmt.Fprintln(os.Stderr, e)
				}
				continue
			}

			out <- row{lat: lat, lon: lon, time: time}
		}
	}()
	return out
}

//We need to eventually turn our rows into a linestring to send to mapbox, so it doesn't matter if we build up the array in memory
//at this point.
func toGroupedRow(in <-chan row) <-chan []row {
	out := make(chan []row)
	go func() {
		defer close(out)

		var last *time.Time
		group := make([]row, 0, 1000)
		for r := range in {
			if last == nil {
				timeCopy := r.time
				last = &timeCopy
			}

			diff := r.time.Sub(*last)

			if diff < 0 {
				diff = diff * -1
			}

			if diff > time.Hour {
				out <- group
				group = make([]row, 0, 1000)
			}

			group = append(group, r)
			timeCopy := r.time
			last = &timeCopy

		}
		out <- group
	}()
	return out
}

type walk struct {
	lineString       geojson
	distanceTraveled float64
	totalTime        time.Duration
}

type geojson struct {
	FeatureType string      `json:"type"`
	Coordinates [][]float64 `json:"coordinates"`
}

func toWalk(in <-chan []row) <-chan walk {
	out := make(chan walk)
	go func() {
		defer close(out)
		for group := range in {
			if len(group) < 2 {
				err := fmt.Errorf("group must have at least 2 rows %v", group)
				fmt.Fprintln(os.Stderr, err)
				continue
			}

			last := group[len(group)-1]
			totalTime := last.time.Sub(group[0].time)

			coords := make([][]float64, len(group))
			lineString := geojson{FeatureType: "LineString", Coordinates: coords}

			for i := range lineString.Coordinates {
				lineString.Coordinates[i] = []float64{group[i].lon, group[i].lat}
			}

			out <- walk{lineString: lineString, distanceTraveled: distanceTraveledInMi(group), totalTime: totalTime}
		}
	}()

	return out
}

func distanceTraveledInMi(group []row) float64 {
	totalDistance := float64(0)
	var last *haversine.Coord

	for _, row := range group {
		if last == nil {
			last = &haversine.Coord{Lat: row.lat, Lon: row.lon}
			continue
		}

		current := haversine.Coord{Lat: row.lat, Lon: row.lon}
		mi, _ := haversine.Distance(*last, current)
		totalDistance += mi
	}

	return totalDistance
}

func toUrl(in <-chan walk, apiToken string) <-chan string {
	out := make(chan string)
	go func() {
		defer close(out)

		for w := range in {
			json, err := json.Marshal(w.lineString)
			if err != nil {
				wrapped := fmt.Errorf("could not marshall json for linestring in walk %v: %w", w, err)
				fmt.Fprintln(os.Stderr, wrapped)
				continue
			}

			escaped := url.PathEscape(string(json))

			urlStr := fmt.Sprintf("https://api.mapbox.com/styles/v1/mapbox/streets-v10/static/geojson(%s)/auto/1024x1024@2x?access_token=%s&logo=false", escaped, apiToken)
			out <- urlStr

		}
	}()
	return out
}

type walkMap struct {
	FileName string `json:"file_name"`
}

func makeMap(in <-chan string) {
	for url := range in {
		go func(url string) {
			devNullLogger := log.New(ioutil.Discard, "", 0)
			c := retryablehttp.NewClient()
			c.Logger = devNullLogger

			resp, err := c.Get(url)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}

			defer resp.Body.Close()

			tmp, err := ioutil.TempFile("", "*.png")
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}

			if _, err := io.Copy(tmp, resp.Body); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}

			output, err := json.Marshal(&walkMap{FileName: tmp.Name()})

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return
			}

			fmt.Println(string(output))
		}(url)
	}
}
