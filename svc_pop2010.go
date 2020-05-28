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
	"time"
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

func mkpyramid(buf [24]int16) pyramid {
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
	var record [151]byte
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
	mpopulation := int32(0)
	fpopulation := int32(0)
	const (
		colid   = 0
		colpop  = colid + 9
		collat  = colpop + 4
		collon  = collat + 8
		colx    = collon + 8
		coly    = colx + 8
		colz    = coly + 8
		colmpop = colz + 8 + 2
		colfpop = colz + 8 + 2 + 2*24
	)
	xs, ys, zs := record[colx:colx+8], record[coly:coly+8], record[colz:colz+8]
	lats, lons := record[collat:collat+8], record[collon:collon+8]
	pops := record[colpop : colpop+4]
	mpops := record[colmpop : colmpop+2]
	fpops := record[colfpop : colfpop+2]
	var mpyr [24]int16
	var fpyr [24]int16
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
		mpop := int16(binary.LittleEndian.Uint16(mpops))
		fpop := int16(binary.LittleEndian.Uint16(fpops))
		population += pop
		mpopulation += int32(mpop)
		fpopulation += int32(fpop)
		//
		for ix := range mpyr {
			mpyr[ix] += int16(binary.LittleEndian.Uint16(record[colmpop+2*ix : colmpop+2*(ix+1)]))
		}
		for ix := range fpyr {
			fpyr[ix] += int16(binary.LittleEndian.Uint16(record[colfpop+2*ix : colfpop+2*(ix+1)]))
		}
		filter2++
	}
	resultx := struct {
		Duration int64   `json:"duration_msec"`
		Distance int64   `json:"distance"`
		Lat      float64 `json:"lat"`
		Lon      float64 `json:"lon"`
		Blocks   int     `json:"blocks"`
		Pop2010  int32   `json:"pop2010"`
		Fpop2010 int32   `json:"pop2010_female"`
		Mpop2010 int32   `json:"pop2010_male"`
		Fpyramid pyramid `json:"ages_female"`
		Mpyramid pyramid `json:"ages_male"`
	}{time.Since(start).Milliseconds(), distance, lat, lon, filter2, population, fpopulation, mpopulation, mkpyramid(fpyr), mkpyramid(mpyr)}
	//
	jresult, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, jresult)
}
