package svc

import (
	"encoding/json"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/reconditematter/rnames"
	"net/http"
	"os"
	"strconv"
	"time"
)

// RandomNames -- configures the service for the router `R`.
func RandomNames(R *mux.Router) {
	R.Handle("/api/reconditematter/randomnames", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(usageRandomNames)))
	R.Handle("/api/reconditematter/randomnames/{count}", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(getnamesb)))
	R.Handle("/api/reconditematter/randomnames/{count}/f", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(getnamesf)))
	R.Handle("/api/reconditematter/randomnames/{count}/m", handlers.LoggingHandler(os.Stderr, http.HandlerFunc(getnamesm)))
}

func usageRandomNames(w http.ResponseWriter, r *http.Request) {
	doc := `
/randomnames/{count} -- returns {count} random names.
/randomnames/{count}/f -- returns {count} random female names.
/randomnames/{count}/m -- returns {count} random male names.

Input:
{count} = 1,...,1000

Output:
{
 "duration_msec":___,
 "count":___,
 "fcount":___,
 "mcount":___,
 "names":[{"family":___,"given":___,"gender":___},...]
}

Data sources:
1000 most popular given names of each gender (2017 US SSA)
1000 most frequent family names (2010 US Census)
`
	//
	HS200t(w, []byte(doc))
}

func getnamesb(w http.ResponseWriter, r *http.Request) {
	getnames(w, r, rnames.GenBoth)
}

func getnamesf(w http.ResponseWriter, r *http.Request) {
	getnames(w, r, rnames.GenF)
}

func getnamesm(w http.ResponseWriter, r *http.Request) {
	getnames(w, r, rnames.GenM)
}

func getnames(w http.ResponseWriter, r *http.Request, gengen int) {
	start := time.Now()
	vars := mux.Vars(r)
	count, err := strconv.ParseInt(vars["count"], 10, 64)
	if err != nil {
		HS400(w)
		return
	}
	//
	result, err := rnames.Gen(int(count), gengen)
	if err != nil {
		HS400(w)
		return
	}
	//
	fcount := 0
	for _, hn := range result {
		if hn.Gender == "female" {
			fcount++
		}
	}
	//
	resultx := struct {
		Duration int64              `json:"duration_msec"`
		Count    int                `json:"count"`
		FCount   int                `json:"fcount"`
		MCount   int                `json:"mcount"`
		Names    []rnames.HumanName `json:"names"`
	}{time.Since(start).Milliseconds(), len(result), fcount, len(result) - fcount, result}
	//
	jresult, err := json.Marshal(resultx)
	if err != nil {
		HS500(w)
		return
	}
	//
	HS200j(w, jresult)
}
