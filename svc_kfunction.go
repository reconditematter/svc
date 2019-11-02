package svc

import (
	"encoding/json"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/reconditematter/ons2"
	"math"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"sync"
	"time"
)

// KFunction -- configures the service for the router `R`.
func KFunction(R *mux.Router) {
	R.Handle("/api/reconditematter/kfunction", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usageKFunction)))
	R.Handle("/api/reconditematter/kfunction/{count}/ncpu/{ncpu}", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(computeKFunction)))
}

func usageKFunction(w http.ResponseWriter, r *http.Request) {
	doc := `
/kfunction/{count}/ncpu/{ncpu} -- computes Ripley's K function for {count} random points on the unit sphere.

Input:
{count} = 2,...,1000
{ncpu} = 1,...,16 -- how many logical CPUs can be executing simultaneously

Output:
{
 "duration_msec":___,
 "count":___,
 "kfunction":[K(0),...,K(180)]
}

The values of K functions are computed as Kripley(t)-Kpois(t), t=0,1,...,180 [deg].
`
	//
	HS200t(w, []byte(doc))
}

func computeKFunction(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	vars := mux.Vars(r)
	count, err := strconv.ParseInt(vars["count"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(2 <= count && count <= 1000) {
		HS400(w)
		return
	}
	//
	ncpu, err := strconv.ParseInt(vars["ncpu"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	if !(1 <= ncpu && ncpu <= 16) {
		HS400(w)
		return
	}
	//
	const D = 181
	var h [D]float64
	compute := func() {
		runtime.GOMAXPROCS(int(ncpu))
		n := int(count)
		points := make([]ons2.Point, n)
		for i := range points {
			points[i] = ons2.Random()
		}
		//
		var wg sync.WaitGroup
		wg.Add(D)
		for k := 0; k < D; k++ {
			go func(k int) {
				θ := math.Pi * float64(k) / 180
				h[k] = ons2.Kripley(points, θ) - ons2.Kpois(θ)
				wg.Done()
			}(k)
		}
		wg.Wait()
	}
	//
	compute()
	//
	resultx := struct {
		Duration  int64      `json:"duration_msec"`
		Count     int        `json:"count"`
		Kfunction [D]float64 `json:"kfunction"`
	}{time.Since(start).Milliseconds(), int(count), h}
	//
	jresult, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, jresult)
}
