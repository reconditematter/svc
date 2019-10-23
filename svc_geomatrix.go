package svc

import (
	"encoding/json"
	"errors"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/reconditematter/cds"
	"github.com/reconditematter/geomys"
	"math"
	"net/http"
	"os"
	"sort"
)

// GeoMatrix -- configures the service for the router `R`.
func GeoMatrix(R *mux.Router) {
	R.Handle("/api/reconditematter/geomatrix", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usageGeoMatrix)))
	R.Handle("/api/reconditematter/geomatrix/compute", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(matcomp)))
	R.Handle("/api/reconditematter/geomatrix/usbig10", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(matbig10)))
	R.Handle("/api/reconditematter/geomatrix/uscap50", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(matcap50)))
	R.Handle("/api/reconditematter/geomatrix/usbig10/sort", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(matbig10s)))
	R.Handle("/api/reconditematter/geomatrix/uscap50/sort", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(matcap50s)))
	R.Handle("/api/reconditematter/geomatrix/compute/sort", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(matcomps)))
}

func usageGeoMatrix(w http.ResponseWriter, r *http.Request) {
	doc := `
/geomatrix/usbig10[/sort] -- computes a matrix of geographic distances between 10 largest US cities.
/geomatrix/uscap50[/sort] -- computes a matrix of geographic distances between 50 US state capitals.
/geomatrix/compute[/sort] -- (POST) computes a matrix of geographic distances between given locations.

[/sort] -- orders the output by geographic distances.

Input:
{
 "ids": ["{id1}","{id2}",...],
 "crd": [{lat1},{lon1},{lat2},{lon2},...]
}

Output:
{
 "count":___,
 "distances":
  [
   {
    "from":___,
    "to":___,
    "km":___,
    "mi":___
   },...
  ]
}
`
	//
	HS200t(w, []byte(doc))
}

// location -- represents a location.
type location struct {
	Id  string  `json:"id"`
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// big10 -- the locations of 10 largest (by population) cities in the US.
var big10 = [...]location{
	location{"New York", 40.6635, -73.9387},
	location{"Los Angeles", 34.0194, -118.4108},
	location{"Chicago", 41.8376, -87.6818},
	location{"Houston", 29.7866, -95.3909},
	location{"Phoenix", 33.5722, -112.0901},
	location{"Philadelphia", 40.0094, -75.1333},
	location{"San Antonio", 29.4724, -98.5251},
	location{"San Diego", 32.8153, -117.1350},
	location{"Dallas", 32.7933, -96.7665},
	location{"San Jose", 37.2967, -121.8189},
}

// cap50 -- the locations of 50 US capital cities.
var cap50 = [...]location{
	location{"Juneau, AK", 58.3815686, -135.3187015},
	location{"Montgomery, AL", 32.3440116, -86.3861224},
	location{"Little Rock, AR", 34.7242069, -92.4780122},
	location{"Phoenix, AZ", 33.6056711, -112.4052389},
	location{"Sacramento, CA", 38.5617255, -121.5829968},
	location{"Denver, CO", 39.7645187, -104.9951956},
	location{"Hartford, CT", 41.7657462, -72.7151064},
	location{"Dover, DE", 39.1565281, -75.5834599},
	location{"Tallahassee, FL", 30.4673606, -84.396941},
	location{"Atlanta, GA", 33.7679192, -84.5606887},
	location{"Honolulu, HI", 21.3281792, -157.8691134},
	location{"Des Moines, IA", 41.5667771, -93.6765559},
	location{"Boise, ID", 43.600909, -116.3039379},
	location{"Springfield, IL", 39.7640172, -89.8109155},
	location{"Indianapolis, IN", 39.7799642, -86.2728342},
	location{"Topeka, KS", 39.0131669, -95.778071},
	location{"Frankfort, KY", 38.1945078, -84.9016449},
	location{"Baton Rouge, LA", 30.4416952, -91.2515036},
	location{"Boston, MA", 42.3145186, -71.1103679},
	location{"Annapolis, MD", 38.9725304, -76.5397139},
	location{"Augusta, ME", 44.3335333, -69.8007319},
	location{"Lansing, MI", 42.7087864, -84.6294674},
	location{"Saint Paul, MN", 44.9398076, -93.1760932},
	location{"Jefferson City, MO", 38.5712792, -92.2324449},
	location{"Jackson, MS", 32.3104541, -90.2638275},
	location{"Helena, MT", 46.5934116, -112.0507134},
	location{"Raleigh, NC", 35.843965, -78.7851405},
	location{"Bismark, ND", 46.8091721, -100.8370943},
	location{"Lincoln, NE", 40.8007178, -96.7607682},
	location{"Concord, NH", 43.2309052, -71.6326453},
	location{"Trenton, NJ", 40.2161138, -74.8092249},
	location{"Santa Fe, NM", 35.6826126, -106.0530761},
	location{"Carson City, NV", 39.1680158, -119.9164112},
	location{"Albany, NY", 42.6681893, -73.8807209},
	location{"Columbus, OH", 39.9831302, -83.1309131},
	location{"Oklahoma City, OK", 35.4828833, -97.7593846},
	location{"Salem, OR", 44.9330916, -123.0982472},
	location{"Harrisburg, PA", 40.2822047, -76.9154449},
	location{"Providence, RI", 41.8170512, -71.4561999},
	location{"Columbia, SC", 34.037714, -81.0776497},
	location{"Pierre, SD", 44.370824, -100.3555579},
	location{"Nashville, TN", 36.1868683, -87.0654323},
	location{"Austin, TX", 30.3079827, -97.8934865},
	location{"Salt Lake City, UT", 40.7767833, -112.0605691},
	location{"Richmond, VA", 37.5247764, -77.5633015},
	location{"Montpelier, VT", 44.274248, -72.6037393},
	location{"Olympia, WA", 47.0393866, -122.928888},
	location{"Madison, WI", 43.0851588, -89.5465042},
	location{"Charleston, WV", 38.3435627, -81.7135836},
	location{"Cheyenne, WY", 41.1476406, -104.8374445},
}

func matbig10(w http.ResponseWriter, r *http.Request) {
	matloc(w, r, big10[:], false)
}

func matbig10s(w http.ResponseWriter, r *http.Request) {
	matloc(w, r, big10[:], true)
}

func matcap50(w http.ResponseWriter, r *http.Request) {
	matloc(w, r, cap50[:], false)
}

func matcap50s(w http.ResponseWriter, r *http.Request) {
	matloc(w, r, cap50[:], true)
}

// tpost -- represents an input for POST.
type tpost struct {
	Ids []string  `json:"ids"`
	Crd []float64 `json:"crd"`
}

func matparse(w http.ResponseWriter, r *http.Request) (t tpost, ok bool) {
	const NMAX = 100
	ok = false
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	err := decoder.Decode(&t)
	if err != nil {
		// JSON error
		HS400t(w, err.Error())
		return
	}
	//
	n := len(t.Ids)
	if n > NMAX || 2*n != len(t.Crd) {
		// array length error
		HS400t(w, "array length error")
		return
	}
	//
	setofid := cds.NewSetOfStr()
	for _, id := range t.Ids {
		setofid.Extend(id)
	}
	if setofid.Card() != n {
		// repeated ids error
		HS400t(w, "repeated ids error")
		return
	}
	//
	ok = true
	return
}

func matcomp(w http.ResponseWriter, r *http.Request) {
	t, ok := matparse(w, r)
	if !ok {
		return
	}
	//
	loc := make([]location, len(t.Ids))
	for i := range loc {
		loc[i] = location{t.Ids[i], t.Crd[2*i], t.Crd[2*i+1]}
	}
	//
	matloc(w, r, loc, false)
}

func matcomps(w http.ResponseWriter, r *http.Request) {
	t, ok := matparse(w, r)
	if !ok {
		return
	}
	//
	loc := make([]location, len(t.Ids))
	for i := range loc {
		loc[i] = location{t.Ids[i], t.Crd[2*i], t.Crd[2*i+1]}
	}
	//
	matloc(w, r, loc, true)
}

type jrep struct {
	From string  `json:"from"`
	To   string  `json:"to"`
	Km   float64 `json:"km"`
	Mi   float64 `json:"mi"`
}

func matloc(w http.ResponseWriter, r *http.Request, loc []location, dosort bool) {
	n := len(loc)
	crd := make([][2]float64, n)
	for i, loci := range loc {
		crd[i][0] = loci.Lat
		crd[i][1] = loci.Lon
	}
	//
	D, err := computegeomat(crd)
	if err != nil {
		HS400t(w, err.Error())
		return
	}
	//
	result := make([]jrep, 0)
	const mifactor = (1200.0 / 3937.0) * 5280.0
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			meters := D[[2]int{i, j}]
			miles := meters / mifactor
			result = append(result, jrep{loc[i].Id, loc[j].Id, math.Round(meters/100) / 10, math.Round(miles*10) / 10})
		}
	}
	//
	resultx := struct {
		Count int    `json:"count"`
		Dist  []jrep `json:"distances"`
	}{len(result), result}
	//
	if dosort {
		sort.Sort(distslice(resultx.Dist))
	}
	//
	resultj, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, resultj)
}

func computegeomat(points [][2]float64) (map[[2]int]float64, error) {
	n := len(points)
	mat := make(map[[2]int]float64)
	wgs1984 := geomys.WGS1984()
	//
	for i, pi := range points {
		lati, loni := pi[0], pi[1]
		if !(-90 <= lati && lati <= 90 && -180 <= loni && loni <= 180) {
			return nil, errors.New("coordinate error")
		}
		//
		p1 := geomys.Geo(lati, loni)
		for j := i + 1; j < n; j++ {
			pj := points[j]
			latj, lonj := pj[0], pj[1]
			if !(-90 <= latj && latj <= 90 && -180 <= lonj && lonj <= 180) {
				return nil, errors.New("coordinate error")
			}
			//
			p2 := geomys.Geo(latj, lonj)
			d := geomys.Andoyer(wgs1984, p1, p2)
			mat[[2]int{i, j}] = d
		}
	}
	//
	return mat, nil
}

// distslice implements sort.Interface
type distslice []jrep

func (s distslice) Len() int {
	return len(s)
}

func (s distslice) Less(i, j int) bool {
	return s[i].Km < s[j].Km
}

func (s distslice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
