package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"time"
)

type bitRequest struct {
	AppID string `json:"app_id"`
	DSPID string `json:"dsp_id"`
}
type bitResponse struct {
	DSPID string `json:"dsp_id"`
	Price int    `json:"price"`
}

type winRequest struct {
	DSPID string `json:"dsp_id"`
	Price int    `json:"price"`
}
type winResponse struct {
	URL string `json:"url"`
}

// DSP DSPのメインロジック
type DSP struct {
}

var rs1Letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func main() {
	rand.Seed(time.Now().Unix())

	addr := ":8080"
	d := &DSP{}
	http.HandleFunc("/", d.BitHandler)
	http.HandleFunc("/win", d.WinHandler)
	log.Printf("DSP Server Listening on " + addr + " ...")
	if err := http.ListenAndServe(addr, nil); err != nil {
	}
}

// BitHandler SSPからのビットリクエストを処理する
func (d *DSP) BitHandler(w http.ResponseWriter, r *http.Request) {
	i := rand.Intn(2000)
	fmt.Println(i)
	time.Sleep(time.Duration(i) * time.Millisecond)

	bitReq := bitRequest{}
	req, err := ioutil.ReadAll(r.Body)
	if err != nil {
	}
	if err := json.Unmarshal(req, &bitReq); err != nil {
	}
	bitRes, err := d.bit(bitReq)
	if err != nil {
	}
	fmt.Printf("(%%#v) %#v\n", bitRes)
	if err := json.NewEncoder(w).Encode(bitRes); err != nil {
	}
}

// WinHandler SSPからのWinリクエストを処理する
func (d *DSP) WinHandler(w http.ResponseWriter, r *http.Request) {
	winReq := winRequest{}
	req, err := ioutil.ReadAll(r.Body)
	if err != nil {
	}
	if err := json.Unmarshal(req, &winReq); err != nil {
	}
	winRes, err := d.win(winReq)
	if err != nil {
	}
	if err := json.NewEncoder(w).Encode(winRes); err != nil {
	}
}

func (d *DSP) bit(bitReq bitRequest) (*bitResponse, error) {
	return &bitResponse{
		DSPID: bitReq.DSPID,
		Price: rand.Intn(100),
	}, nil
}

func (d *DSP) win(winReq winRequest) (*winResponse, error) {
	url := "http://" + randString(10) + ".com"

	return &winResponse{
		URL: url,
	}, nil
}

func randString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = rs1Letters[rand.Intn(len(rs1Letters))]
	}
	return string(b)
}
