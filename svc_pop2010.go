package svc

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/reconditematter/geomys"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
)

// Pop2010 -- configures the service for the router `R`.
func Pop2010(R *mux.Router) {
	R.Handle("/api/pop2010", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usagePop2010)))
	R.Handle("/api/pop2010/{distance}/lat/{lat}/lon/{lon}", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(pop2010)))
}

func usagePop2010(w http.ResponseWriter, r *http.Request) {
	doc := `
/api/pop2010/{distance}/lat/{lat}/lon/{lon} -- returns the population (US Census 2010) within the given distance from the given location.

Input:
{distance} -- the search radius in meters, must be in [1,1000000]
{lat} -- the geographic latitude, must be in [-90,90]
{lon} -- the geographic longitude, must be in [-180,180]

Output:
{
 "distance":___,
 "lat":___,
 "lon":___,
 "blocks":___,
 "pop2010":___
}

{blocks} -- US Census block count within the given distance
`
	//
	HS200t(w, []byte(doc))
}

func pop2010(w http.ResponseWriter, r *http.Request) {
	round := func(x float64) int64 {
		y := math.Ceil(math.Abs(x))
		if x < 0 {
			return int64(-y)
		}
		return int64(y)
	}
	//
	const Pop2010File = "nozgeo.bin"
	vars := mux.Vars(r)
	distance, err := strconv.ParseInt(vars["distance"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(1 <= distance && distance <= 1000000) {
		HS400(w)
		return
	}
	//
	lat, err := strconv.ParseFloat(vars["lat"], 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(-90 <= lat && lat <= 90) {
		HS400(w)
		return
	}
	//
	lon, err := strconv.ParseFloat(vars["lon"], 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(-180 <= lon && lon <= 180) {
		HS400(w)
		return
	}
	//
	infile, err := os.Open(Pop2010File)
	if err != nil {
		HS500(w)
		return
	}
	defer infile.Close()
	//
	rdr := bufio.NewReader(infile)
	var record [53]byte
	buf := record[:]
	wgs1984 := geomys.WGS1984()
	geocen := geomys.NewGeocentric(wgs1984)
	//
	query := geomys.Geo(lat, lon)
	dist := float64(distance)
	qxyz := geocen.Forward(query)
	xmin, xmax := round(qxyz[0]-dist), round(qxyz[0]+dist)
	ymin, ymax := round(qxyz[1]-dist), round(qxyz[1]+dist)
	zmin, zmax := round(qxyz[2]-dist), round(qxyz[2]+dist)
	filter2 := 0
	population := int32(0)
	const (
		colid  = 0
		colpop = colid + 9
		collat = colpop + 4
		collon = collat + 8
		colx   = collon + 8
		coly   = colx + 8
		colz   = coly + 8
	)
	xs, ys, zs := record[colx:colx+8], record[coly:coly+8], record[colz:colz+8]
	lats, lons := record[collat:collat+8], record[collon:collon+8]
	pops := record[colpop : colpop+4]
	for {
		_, err := io.ReadFull(rdr, buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			HS500(w)
			return
		}
		//
		x := int64(binary.LittleEndian.Uint64(xs))
		if !(xmin <= x && x <= xmax) {
			continue
		}
		y := int64(binary.LittleEndian.Uint64(ys))
		if !(ymin <= y && y <= ymax) {
			continue
		}
		z := int64(binary.LittleEndian.Uint64(zs))
		if !(zmin <= z && z <= zmax) {
			continue
		}
		//
		lat := math.Float64frombits(binary.LittleEndian.Uint64(lats))
		lon := math.Float64frombits(binary.LittleEndian.Uint64(lons))
		//
		point := geomys.Geo(lat, lon)
		if geomys.Andoyer(wgs1984, query, point) > dist {
			continue
		}
		pop := int32(binary.LittleEndian.Uint32(pops))
		population += pop
		filter2++
	}
	resultx := struct {
		Distance int64   `json:"distance"`
		Lat      float64 `json:"lat"`
		Lon      float64 `json:"lon"`
		Blocks   int     `json:"blocks"`
		Pop2010  int32   `json:"pop2010"`
	}{distance, lat, lon, filter2, population}
	//
	jresult, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, jresult)
}
