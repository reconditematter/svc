// Copyright (c) 2019-2020 Leonid Kneller. All rights reserved.
// Licensed under the MIT license.
// See the LICENSE file for full license information.

package svc

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"github.com/dgraph-io/badger"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/reconditematter/geomys"
	"io"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Pop2010 -- configures the service for the router `R`.
func Pop2010(R *mux.Router) {
	R.Handle("/api/pop2010", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usagePop2010))).Methods("GET")
	R.Handle("/api/pop2010/{distance}/lat/{lat}/lon/{lon}", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(pop2010))).Methods("GET")
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
 "duration_msec":___,
 "distance":___,
 "lat":___,
 "lon":___,
 "blocks":___,
 "pop2010":___,
 "pop2010_female":___,
 "pop2010_male":___,
 "ages_female":{"age_under5":___,"age_5to9":___,...,"age_85over":___},
 "ages_male":{"age_under5":___,"age_5to9":___,...,"age_85over":___}
}

{blocks} -- US Census block count within the given distance
`
	//
	HS200t(w, []byte(doc))
}

type poploc struct {
	id       string
	pop      int
	lat, lon float64
	x, y, z  int
}

var poplocs []poploc
var popbddb *badger.DB

const geofilename = "nozgeo.txt"
const popbddbname = "./bddb"

func init() {
	var err error
	poplocs = loadpoplocs(geofilename)
	popbddb, err = badger.Open(badger.DefaultOptions(popbddbname).WithReadOnly(true).WithLoggingLevel(2))
	if err != nil {
		panic(err)
	}
}

func loadpoplocs(name string) []poploc {
	file, err := os.Open(name)
	if err != nil {
		panic(err)
	}
	rdr := csv.NewReader(bufio.NewReader(file))
	locs := make([]poploc, 0)
	for {
		rec, err := rdr.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		//
		loc := poploc{rec[0], musti(rec[1]), mustf(rec[2]), mustf(rec[3]), musti(rec[4]), musti(rec[5]), musti(rec[6])}
		locs = append(locs, loc)
	}
	//
	return locs
}

func mustf(s string) float64 {
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		panic(err)
	}
	return f
}

func musti(s string) int {
	f, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return int(f)
}

func round(x float64) int {
	y := math.Ceil(math.Abs(x))
	if x < 0 {
		return int(-y)
	}
	return int(y)
}

func geosearch(locs []poploc, query geomys.Point, dist float64) []string {
	spheroid := geomys.WGS1984()
	geocen := geomys.NewGeocentric(spheroid)
	xyz := geocen.Forward(query)
	xmin, xmax := round(xyz[0]-dist), round(xyz[0]+dist)
	ymin, ymax := round(xyz[1]-dist), round(xyz[1]+dist)
	zmin, zmax := round(xyz[2]-dist), round(xyz[2]+dist)
	//
	ids := make([]string, 0)
	for _, loc := range locs {
		if !(xmin <= loc.x && loc.x <= xmax) {
			continue
		}
		if !(ymin <= loc.y && loc.y <= ymax) {
			continue
		}
		if !(zmin <= loc.z && loc.z <= zmax) {
			continue
		}
		//
		if geomys.Andoyer(spheroid, query, geomys.Geo(loc.lat, loc.lon)) <= dist {
			ids = append(ids, loc.id)
		}
	}
	//
	return ids
}

func popsearch(db *badger.DB, keys []string) ([]string, error) {
	vals := make([]string, len(keys))
	//
	err := db.View(func(txn *badger.Txn) error {
		for k, key := range keys {
			item, err := txn.Get([]byte(key))
			if err != nil {
				return err
			}
			val, err := item.ValueCopy(nil)
			if err != nil {
				return err
			}
			vals[k] = string(val)
		}
		return nil
	})
	//
	if err != nil {
		return nil, err
	}
	//
	return vals, nil
}

func popsummary(recs []string) (pop, mpop, fpop int, mpyr, fpyr [24]int) {
	for _, rec := range recs {
		r := strings.Split(rec, ",")
		pop += musti(r[0])
		mpop += musti(r[1])
		fpop += musti(r[25])
		for k := range mpyr {
			mpyr[k] += musti(r[k+1])
			fpyr[k] += musti(r[k+25])
		}
	}
	return
}

type pyramid struct {
	Age00to04 int `json:"age_under5"`
	Age05to09 int `json:"age_5to9"`
	Age10to14 int `json:"age_10to14"`
	Age15to17 int `json:"age_15to17"`
	Age18to19 int `json:"age_18to19"`
	Age20     int `json:"age_20"`
	Age21     int `json:"age_21"`
	Age22to24 int `json:"age_22to24"`
	Age25to29 int `json:"age_25to29"`
	Age30to34 int `json:"age_30to34"`
	Age35to39 int `json:"age_35to39"`
	Age40to44 int `json:"age_40to44"`
	Age45to49 int `json:"age_45to49"`
	Age50to54 int `json:"age_50to54"`
	Age55to59 int `json:"age_55to59"`
	Age60to61 int `json:"age_60to61"`
	Age62to64 int `json:"age_62to64"`
	Age65to66 int `json:"age_65to66"`
	Age67to69 int `json:"age_67to69"`
	Age70to74 int `json:"age_70to74"`
	Age75to79 int `json:"age_75to79"`
	Age80to84 int `json:"age_80to84"`
	Age85over int `json:"age_85over"`
}

func mkpyramid(buf [24]int) pyramid {
	var pyr pyramid
	// ignore buf[0]
	pyr.Age00to04 = int(buf[1])
	pyr.Age05to09 = int(buf[2])
	pyr.Age10to14 = int(buf[3])
	pyr.Age15to17 = int(buf[4])
	pyr.Age18to19 = int(buf[5])
	pyr.Age20 = int(buf[6])
	pyr.Age21 = int(buf[7])
	pyr.Age22to24 = int(buf[8])
	pyr.Age25to29 = int(buf[9])
	pyr.Age30to34 = int(buf[10])
	pyr.Age35to39 = int(buf[11])
	pyr.Age40to44 = int(buf[12])
	pyr.Age45to49 = int(buf[13])
	pyr.Age50to54 = int(buf[14])
	pyr.Age55to59 = int(buf[15])
	pyr.Age60to61 = int(buf[16])
	pyr.Age62to64 = int(buf[17])
	pyr.Age65to66 = int(buf[18])
	pyr.Age67to69 = int(buf[19])
	pyr.Age70to74 = int(buf[20])
	pyr.Age75to79 = int(buf[21])
	pyr.Age80to84 = int(buf[22])
	pyr.Age85over = int(buf[23])
	return pyr
}

func pop2010(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	//
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
	keys := geosearch(poplocs, geomys.Geo(lat, lon), float64(distance))
	recs, err := popsearch(popbddb, keys)
	if err != nil {
		HS500(w)
		return
	}
	//
	population, mpopulation, fpopulation, mpyr, fpyr := popsummary(recs)
	//
	resultx := struct {
		Duration int64   `json:"duration_msec"`
		Distance int64   `json:"distance"`
		Lat      float64 `json:"lat"`
		Lon      float64 `json:"lon"`
		Blocks   int     `json:"blocks"`
		Pop2010  int     `json:"pop2010"`
		Fpop2010 int     `json:"pop2010_female"`
		Mpop2010 int     `json:"pop2010_male"`
		Fpyramid pyramid `json:"ages_female"`
		Mpyramid pyramid `json:"ages_male"`
	}{time.Since(start).Milliseconds(), distance, lat, lon, len(recs), population, fpopulation, mpopulation, mkpyramid(fpyr), mkpyramid(mpyr)}
	//
	jresult, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, jresult)
}
