// Copyright (c) 2019-2021 Leonid Kneller. All rights reserved.
// Licensed under the MIT license.
// See the LICENSE file for full license information.

package svc

import (
	"encoding/json"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/reconditematter/geomys"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

// GeoCircle -- configures the service for the router `R`.
func GeoCircle(R *mux.Router) {
	R.Handle("/api/geocircle", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usageGeoCircle))).Methods("GET")
	R.Handle("/api/geocircle/{level}/lat/{lat}/lon/{lon}/radius/{radius}", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(geocircle))).Methods("GET")
}

func usageGeoCircle(w http.ResponseWriter, r *http.Request) {
	doc := `
/api/geocircle/{level}/lat/{lat}/lon/{lon}/radius/{radius} -- generates a geodesic circle around a given geographic location.
 
Input:
{level} = 1,...,4 -- the level of hierarchy (1=360 points,...,4=2880 points)
{lat} -- the geographic latitude of the center, must be in [-90,90]
{lon} -- the geographic longitude of the center, must be in [-180,180]
{radius} -- the circle radius in meters, must be in [1,1000000]
 
Output:
{
 "duration_ms":___,
 "type":"GeodesicCircle",
 "center":{"lat":___,"lon":___},
 "radius":___,
 "count":___,
 "path":[{"lat":___,"lon":___},...]
}
`
	//
	HS200t(w, []byte(doc))
}

func geocircle(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	//
	level, err := strconv.ParseInt(vars["level"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(1 <= level && level <= 4) {
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
	radius, err := strconv.ParseInt(vars["radius"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(1 <= radius && radius <= 1000000) {
		HS400(w)
		return
	}
	//
	type geo2 struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	type geopath2 []geo2
	//
	round9 := func(x float64) float64 {
		y := int64(math.Abs(x)*1000000000 + 0.5)
		if x < 0 {
			y = -y
		}
		return float64(y) / 1000000000
	}
	//
	C := gengeocircle(geomys.Geo(lat, lon), float64(radius), int(level))
	result := make(geopath2, len(C))
	for k, pk := range C {
		lat, lon := pk.Geo()
		result[k] = geo2{Lat: round9(lat), Lon: round9(lon)}
	}
	//
	resultx := struct {
		Duration int64    `json:"duration_ms"`
		Type     string   `json:"type"`
		Center   geo2     `json:"center"`
		Radius   int64    `json:"radius"`
		Count    int      `json:"count"`
		Path     geopath2 `json:"path"`
	}{time.Since(start).Milliseconds(), "GeodesicCircle", geo2{round9(lat), round9(lon)}, radius, len(result), result}
	//
	jresult, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, jresult)
}

func gengeocircle(c geomys.Point, s float64, level int) (ps []geomys.Point) {
	if level < 1 {
		level = 1
	}
	if level > 4 {
		level = 4
	}
	var n int
	switch level {
	case 1:
		n = 360
	case 2:
		n = 720
	case 3:
		n = 1440
	case 4:
		n = 2880
	}
	//
	step := 360.0 / float64(n)
	ps = make([]geomys.Point, n+1)
	G := geomys.NewGeodesic(geomys.WGS1984())
	for k := 0; k < n; k++ {
		alpha := float64(k) * step
		if alpha > 180 {
			alpha -= 360
		}
		p, _ := G.Direct(c, alpha, s)
		ps[k] = p
	}
	ps[n] = ps[0]
	return
}
