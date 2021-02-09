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
)

// GeoHash -- configures the service for the router `R`.
func GeoHash(R *mux.Router) {
	R.Handle("/api/geohash", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usageGeoHash))).Methods("GET")
	R.Handle("/api/geohash/{length}/lat/{lat}/lon/{lon}", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(geohash))).Methods("GET")
}

// HashGeo -- configures the service for the router `R`.
func HashGeo(R *mux.Router) {
	R.Handle("/api/hashgeo", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usageHashGeo))).Methods("GET")
	R.Handle("/api/hashgeo/{hash}", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(hashgeo))).Methods("GET")
}

func usageGeoHash(w http.ResponseWriter, r *http.Request) {
	doc := `
/api/geohash/{length}/lat/{lat}/lon/{lon} -- returns the geohash of the given {length} for the geographic coordinates {lat},{lon}.

Input:
{length} = 3,5,7,9,11,13,15 -- the length of the computed geohash
{lat} -- the geographic latitude, must be in [-90,90]
{lon} -- the geographic longitude, must be in [-180,180]

Output:
{
 "lat":___,
 "lon":___,
 "geohash":___,
 "res_d":___,
 "res_m":___
}

{res_d} -- the resolution of the returned geohash measured in degrees
{res_m} -- the resolution of the returned geohash measured in meters
           (on the equator, assuming the equatorial radius 6378137 m)
`
	//
	HS200t(w, []byte(doc))
}

func usageHashGeo(w http.ResponseWriter, r *http.Request) {
	doc := `
/api/hashgeo/{hash} -- returns the geographic coordinates encoded in the given {hash}.

Input:
{hash} -- the geohash to decode

Output:
{
 "lat":___,
 "lon":___,
 "geohash":___
}
`
	//
	HS200t(w, []byte(doc))
}

func geohash(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//
	length, err := strconv.ParseInt(vars["length"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(length == 3 || length == 5 || length == 7 || length == 9 || length == 11 || length == 13 || length == 15) {
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
	hash, resd := geomys.GeoHash(int(length), geomys.Geo(lat, lon))
	// WGS1984 equatorial radius
	resm := 2 * math.Pi * 6378137 * (resd / 360)
	//
	resultx := struct {
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
		Geohash string  `json:"geohash"`
		Resd    float64 `json:"res_d"`
		Resm    float64 `json:"res_m"`
	}{lat, lon, hash, resd, resm}
	//
	jresult, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, jresult)
}

func hashgeo(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//
	hash := vars["hash"]
	if len(hash) < 3 {
		HS400(w)
		return
	}
	if len(hash) > 15 {
		hash = hash[0:15]
	}
	//
	p, ok := geomys.HashGeo(hash)
	if !ok {
		HS400(w)
		return
	}
	//
	lat, lon := p.Geo()
	resultx := struct {
		Lat     float64 `json:"lat"`
		Lon     float64 `json:"lon"`
		Geohash string  `json:"geohash"`
	}{lat, lon, hash}
	//
	jresult, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, jresult)
}
