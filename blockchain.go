package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Block struct {
	ParentHeader string    `json:"parentHeader"`
	Header       string    `json:"header"`
	Signature    string    `json:"signature"`
	Number       int       `json:"number"`
	Timestamp    time.Time `json:"timestamp"`
}

type Authorities []string

const (
	stepDuration = 5
	n            = 7
)

var BlockChain = map[string]Block{}

func main() {
	// authorities := []string{"divya", "jaz", "loong", "noah", "ross", "susruth", "yunshi"}

	lastStep := int32(0)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {

		defer wg.Done()

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {

			case <-ticker.C:
				t := int32(time.Now().Unix())
				if (t/stepDuration)%n == 0 {
					generateBlock()
				}

			}
		}
	}()
	go func() {
		defer wg.Done()

		ticker := time.NewTicker(time.Millisecond)
		defer ticker.Stop()

		for {
			select {

			case <-ticker.C:
				t := int32(time.Now().Unix())
				if lastStep != (t/stepDuration)%n {
					lastStep = (t / stepDuration) % n
					propogateBlocks()
				}

			}
		}
	}()

	log.Printf("listening at 0.0.0.0:%v...", os.Getenv("PORT"))
	if err := http.ListenAndServe(fmt.Sprintf("0.0.0.0:%v", os.Getenv("PORT")), NewServer()); err != nil {
		log.Fatalf("error listening and serving: %v", err)
	}

	wg.Wait()
}

func NewServer() http.Handler {
	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/block", PostHandler()).Methods("POST")
	r.HandleFunc("/blocks/{title}/page/{page}", GetHandler()).Methods("GET")
	r.Use(RecoveryHandler)

	handler := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST"},
	}).Handler(r)

	return handler
}

// PostHandler handles all HTTP POST requests
func PostHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		postRequest := Block{}
		if err := json.NewDecoder(r.Body).Decode(&postRequest); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(fmt.Sprintf("cannot decode json: %v", err)))
			return
		}
		if err := HandleBlock(postRequest); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(fmt.Sprintf("cannot open order: %v", err)))
			return
		}
		w.WriteHeader(http.StatusOK)
	}
}

// GetHandler will handle GET requests
func GetHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// vars := mux.Vars(r)

		// offset := vars["offset"]
		// limit := vars["limit"]

		var offset, limit int
		var err error

		values := r.URL.Query()
		vals, ok := values["offset"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("cannot read offset"))
			return
		}

		if len(vals) >= 1 {
			offset, err = strconv.Atoi(vals[0])
		}

		vals, ok = values["limit"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("cannot read limit"))
			return
		}

		if len(vals) >= 1 {
			limit, err = strconv.Atoi(vals[0])
		}

		blocks := GetBlocks(offset, limit)

		str, err := json.Marshal(blocks)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte("cannot marshal blocks"))
			return
		}

		// Set content type to JSON before StatusOK or it will be ignored
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(str)
	}
}

// RecoveryHandler handles errors while processing the requests and populates the errors in the response
func RecoveryHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if r := recover(); r != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(fmt.Sprintf("%v", r)))
			}
		}()
		h.ServeHTTP(w, r)
	})
}

func HandleBlock(block Block) error {
	return nil
}

func GetBlocks(offset, limit int) []Block {
	return []Block{}
}

func generateBlock() {

	// generate block
	// send to others
	// replace longest tip
}

func propogateBlocks() {

}
