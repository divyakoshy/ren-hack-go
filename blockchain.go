package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
)

type Block struct {
	ParentHeader string `json:"parentHeader"`
	Header       string `json:"header"`
	Signature    string `json:"signature"`
	Number       int    `json:"number"`
	Timestamp    int    `json:"timestamp"`
	BlockNumber  int    `json:"-"`
}

type Authorities []string

const (
	stepDuration = 5
	n            = 4
)

var genesisBlock = Block{
	Header:       "AAAAAAAAAAAAAAAAAAAAAA==",
	ParentHeader: "",
	BlockNumber:  1,
	Signature:    "",
	Timestamp:    0,
	Number:       0,
}

var BlockChain = map[string]Block{}
var longestBlockHeader = genesisBlock.Header
var longestBlockTip = 1

var currentRound = 1

var authorities = []string{"localhost", "10.1.1.153:8080", "10.1.0.178:8000"}

func main() {
	// authorities := []string{"divya", "loong", "susruth", "yunshi"}

	BlockChain[genesisBlock.Header] = genesisBlock

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
				// log.Println("generating block")
				t := int32(time.Now().Unix())
				if (t/stepDuration)%n == 0 {
					generateBlock(currentRound)
					currentRound += n
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

					// log.Println("step has changed, propogating blocks")
					lastStep = (t / stepDuration) % n
					propogateBlocks()
				}

			}
		}
	}()

	log.Println("listening at 0.0.0.0:29177...")
	if err := http.ListenAndServe(":29177", NewServer()); err != nil {
		log.Fatalf("error listening and serving: %v", err)
	}

	wg.Wait()
}

func NewServer() http.Handler {

	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/blocks", PostHandler()).Methods("POST")
	r.HandleFunc("/blocks", GetHandler()).Methods("GET")
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
		postRequest := []Block{}
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

func HandleBlock(blocks []Block) error {
	log.Println("got blocks")
	for _, block := range blocks {
		if _, ok := BlockChain[block.Header]; ok {
			continue
		}
		blocknum := block.Number

		longestBlockTip++
		if blocknum >= longestBlockTip-1 {
			longestBlockHeader = block.Header
			block.BlockNumber = longestBlockTip
			BlockChain[block.Header] = block
			continue
		}

		for header, blk := range BlockChain {
			if blk.BlockNumber >= blocknum {
				blk.BlockNumber++
				BlockChain[header] = blk
			}
		}

		block.BlockNumber = blocknum
		BlockChain[block.Header] = block
	}
	return nil
}

func GetBlocks(offset, limit int) []Block {
	blocks := []Block{}
	for _, block := range BlockChain {
		if block.BlockNumber >= offset && block.BlockNumber < limit {
			blocks = append(blocks, block)
		}
	}
	return blocks
}

func generateBlock(currentRound int) error {

	// generate new block and update longestBlockTip
	_ = createNewBlockAndUpdateBlockchain(currentRound)
	blocks := GetBlocks(0, len(BlockChain))
	// blocks := []Block{block}
	marshalledBlocks, err := json.Marshal(blocks)
	if err != nil {
		return err
	}

	buf := bytes.NewReader(marshalledBlocks)

	// send to others
	for _, auth := range authorities {

		url := fmt.Sprintf("http://" + auth + "/blocks")
		res, err := http.DefaultClient.Post(url, "application/json", buf)
		if err != nil {
			// return err
			return err
		}
		defer res.Body.Close()

		log.Printf("status: %v", res.StatusCode)
		resText, err := ioutil.ReadAll(res.Body)
		if err != nil {
			// return err
			return err
		}
		log.Printf("body: %v", string(resText))
	}
	return nil
}

func propogateBlocks() {

}

func createNewBlockAndUpdateBlockchain(currentRound int) Block {
	header := base64.StdEncoding.EncodeToString(randomBytes())
	var parentHeader string
	blockNum := 0

	if lastBlock, ok := BlockChain[longestBlockHeader]; ok {
		parentHeader = lastBlock.Header
		blockNum = lastBlock.BlockNumber + 1
	}
	block := Block{
		Header:       header,
		ParentHeader: parentHeader,
		BlockNumber:  blockNum,
		Signature:    header + ":divya",
		Timestamp:    int(time.Now().Unix()),
		Number:       currentRound,
	}

	BlockChain[block.Header] = block
	longestBlockHeader = block.Header
	longestBlockTip = block.BlockNumber

	return block

}

// random32Bytes creates a random [32]byte.
func randomBytes() []byte {
	key := make([]byte, 64)
	_, err := rand.Read(key)
	if err != nil {
		// handle error here
	}
	return key
}
