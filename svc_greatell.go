// Copyright (c) 2019-2020 Leonid Kneller. All rights reserved.
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

// GreatEll -- configures the service for the router `R`.
func GreatEll(R *mux.Router) {
	R.Handle("/api/greatell", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usageGreatEll))).Methods("GET")
	R.Handle("/api/greatell/{count}/lat1/{lat1}/lon1/{lon1}/lat2/{lat2}/lon2/{lon2}", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(greatell))).Methods("GET")
}

func usageGreatEll(w http.ResponseWriter, r *http.Request) {
	doc := `
/api/greatell/{count}/lat1/{lat1}/lon1/{lon1}/lat2/{lat2}/lon2/{lon2} -- generates a path along the great ellipse between two given geographic locations.

Input:
{count} = 3,...,1001 -- the number of points in the generated path
{lat1} -- the geographic latitude of the source, must be in [-90,90]
{lon1} -- the geographic longitude of the source, must be in [-180,180]
{lat2} -- the geographic latitude of the target, must be in [-90,90]
{lon2} -- the geographic longitude of the target, must be in [-180,180]

Output:
{
 "duration_ms":___,
 "type":"GreatEllipse",
 "source":{"lat":___,"lon":___},
 "target":{"lat":___,"lon":___},
 "count":___,
 "distance":___,
 "step":___,
 "path":[{"lat":___,"lon":___,"azi":___},...]
}

{distance} -- the distance between the source and the target points in meters
{step} -- the distance between two consecutive points on the path in meters
`
	//
	HS200t(w, []byte(doc))
}

func greatell(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	//
	count, err := strconv.ParseInt(vars["count"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(3 <= count && count <= 1001) {
		HS400(w)
		return
	}
	//
	lat1, err := strconv.ParseFloat(vars["lat1"], 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(-90 <= lat1 && lat1 <= 90) {
		HS400(w)
		return
	}
	//
	lon1, err := strconv.ParseFloat(vars["lon1"], 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(-180 <= lon1 && lon1 <= 180) {
		HS400(w)
		return
	}
	//
	lat2, err := strconv.ParseFloat(vars["lat2"], 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(-90 <= lat2 && lat2 <= 90) {
		HS400(w)
		return
	}
	//
	lon2, err := strconv.ParseFloat(vars["lon2"], 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(-180 <= lon2 && lon2 <= 180) {
		HS400(w)
		return
	}
	//
	type geo2 struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
	}
	type geo3 struct {
		Lat float64 `json:"lat"`
		Lon float64 `json:"lon"`
		Azi float64 `json:"azi"`
	}
	type geopath []geo3
	//
	round3 := func(x float64) float64 {
		y := int64(math.Abs(x)*1000 + 0.5)
		if x < 0 {
			y = -y
		}
		return float64(y) / 1000
	}
	round6 := func(x float64) float64 {
		y := int64(math.Abs(x)*1000000 + 0.5)
		if x < 0 {
			y = -y
		}
		return float64(y) / 1000000
	}
	//
	result := make(geopath, 0)
	source := geomys.Geo(lat1, lon1)
	target := geomys.Geo(lat2, lon2)
	sph := geomys.WGS1984()
	ell := geomys.NewGreatEllipse(sph)
	s12, azi1, azi2 := ell.Inverse(source, target)
	//
	result = append(result, geo3{round6(lat1), round6(lon1), round6(azi1)})
	//
	step := s12 / float64(count-1)
	for k := 1; k < int(count)-1; k++ {
		loc, azi := ell.Direct(source, azi1, float64(k)*step)
		t1, t2 := loc.Geo()
		result = append(result, geo3{round6(t1), round6(t2), round6(azi)})
	}
	//
	result = append(result, geo3{round6(lat2), round6(lon2), round6(azi2)})
	//
	resultx := struct {
		Duration int64   `json:"duration_ms"`
		Type     string  `json:"type"`
		Source   geo2    `json:"source"`
		Target   geo2    `json:"target"`
		Count    int     `json:"count"`
		Distance float64 `json:"distance"`
		Step     float64 `json:"step"`
		Path     geopath `json:"path"`
	}{time.Since(start).Milliseconds(), "GreatEllipse", geo2{result[0].Lat, result[0].Lon}, geo2{result[len(result)-1].Lat, result[len(result)-1].Lon}, len(result), round3(s12), round3(step), result}
	//
	jresult, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, jresult)
}
