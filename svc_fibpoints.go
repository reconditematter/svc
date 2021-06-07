// Copyright (c) 2019-2021 Leonid Kneller. All rights reserved.
// Licensed under the MIT license.
// See the LICENSE file for full license information.

package svc

import (
	"encoding/json"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/reconditematter/ons2"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"
)

// FibPoints -- configures the service for the router `R`.
func FibPoints(R *mux.Router) {
	R.Handle("/api/fibpoints", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usageFibPoints))).Methods("GET")
	R.Handle("/api/fibpoints/{count}/lat/{lat}/lon/{lon}", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(fibPoints))).Methods("GET")
}

func usageFibPoints(w http.ResponseWriter, r *http.Request) {
	doc := `
/api/fibpoints/{count}/lat/{lat}/lon/{lon} -- returns _approximately_ {count} Fibonacci spiral points in a geographic cell [{lat},{lat}+1]x[{lon},{lon}+1].

Input:
{count} = 1,...,1000
{lat} = -90,...,89
{lon} = -180,...,179

Output:
{
 "duration_ms":___,
 "min":{"lat":___,"lon":___},
 "max":{"lat":___,"lon":___},
 "count":___,
 "points":[{"lat":___,"lon":___},...]
}
`
	//
	HS200t(w, []byte(doc))
}

func fibPoints(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	//
	count, err := strconv.ParseInt(vars["count"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(1 <= count && count <= 1000) {
		HS400(w)
		return
	}
	lat, err := strconv.ParseInt(vars["lat"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(-90 <= lat && lat < 90) {
		HS400(w)
		return
	}
	lon, err := strconv.ParseInt(vars["lon"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(-180 <= lon && lon < 180) {
		HS400(w)
		return
	}
	//
	result := ons2.CellFib1x1(int(lat), int(lon), int(count))
	type latlon struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	resultx := struct {
		Duration int64    `json:"duration_ms"`
		Min      latlon   `json:"min"`
		Max      latlon   `json:"max"`
		Count    int64    `json:"count"`
		Points   []latlon `json:"points"`
	}{time.Since(start).Milliseconds(), latlon{float64(lat), float64(lon)}, latlon{float64(lat + 1), float64(lon + 1)}, int64(len(result)), make([]latlon, len(result))}
	for k, p := range result {
		lat, lon := p.Geo()
		resultx.Points[k] = latlon{math.Round(lat*1e8) / 1e8, math.Round(lon*1e8) / 1e8}
	}
	resultj, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, resultj)
}
