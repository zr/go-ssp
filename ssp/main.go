package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

type adRequest struct {
	AppID string `json:"app_id"`
}
type adResponse struct {
	URL string `json:"url"`
}

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

// SSP メインロジックの構造体
type SSP struct {
	bitResponses *[]bitResponse
}

type dspInfo struct {
	bitURL string
	winURL string
}

func main() {
	addr := ":8000"
	s := &SSP{}
	http.HandleFunc("/", s.AdHandler)
	log.Printf("SSP Server Listening on " + addr + " ...")
	if err := http.ListenAndServe(addr, nil); err != nil {
		// Todo: 500を返す
	}
}

// AdHandler ブラウザからの広告取得リクエストを処理する
func (s *SSP) AdHandler(w http.ResponseWriter, r *http.Request) {
	adReq := adRequest{}
	req, err := ioutil.ReadAll(r.Body)
	if err != nil {
	}
	if err := json.Unmarshal(req, &adReq); err != nil {
	}
	adRes, err := s.run(adReq)
	if err != nil {
	}
	if err := json.NewEncoder(w).Encode(adRes); err != nil {
	}
}

// SSPメインロジック
func (s *SSP) run(adReq adRequest) (adResponse, error) {
	// 1.DSPにgoroutineでリクエストを送る
	hosts := s.loadHosts()
	auction := []bitResponse{}

	bitCh := make(chan bitResponse, 0)
	for DSPID, host := range hosts {
		bitRequest := &bitRequest{
			AppID: adReq.AppID,
			DSPID: DSPID,
		}
		go s.sendBit(bitCh, host.bitURL, bitRequest)
	}

	// Todo: contextか何かでエラー処理
	for range hosts {
		select {
		case bitRes, ok := <-bitCh:
			if ok {
				auction = append(auction, bitRes)
			}
		}
	}
	close(bitCh)

	// Todo: auctionがない(DSPがひとつもない)場合

	var firstPrice int
	var secondPrice int
	var winner bitResponse

	// 2. セカンドプライスオークションをする
	for _, bitRes := range auction {
		if firstPrice <= bitRes.Price {
			winner = bitRes
			firstPrice = bitRes.Price
		}
		if secondPrice <= bitRes.Price && !(firstPrice == secondPrice) {
			secondPrice = bitRes.Price
		}
	}

	winReq := &winRequest{
		DSPID: winner.DSPID,
		Price: secondPrice,
	}

	// 3. Win通知を送り、URLを受け取る
	winCh := make(chan winResponse, 0)
	go s.sendWin(winCh, hosts[winner.DSPID].winURL, winReq)
	winRes, ok := <-winCh
	if !ok {
	}
	close(winCh)

	fmt.Println(winRes)

	res := &adResponse{
		URL: winRes.URL,
	}
	return *res, nil
}

func (s *SSP) loadHosts() map[string]dspInfo {
	baseURL := "http://localhost:8080"
	dsp := dspInfo{
		bitURL: baseURL + "/",
		winURL: baseURL + "/win",
	}
	dspMap := map[string]dspInfo{
		"1": dsp,
		"2": dsp,
		"3": dsp,
	}
	return dspMap
}

// sendBit DSPに対してbitリクエストを送る
func (s *SSP) sendBit(ch chan bitResponse, url string, bitReq *bitRequest) {
	var bitRes bitResponse
	if err := sendReq(url, bitReq, &bitRes); err != nil {
	}
	ch <- bitRes
}

// sendWin DSPに対してwinリクエストを送る
func (s *SSP) sendWin(ch chan winResponse, url string, winReq *winRequest) {
	var winRes winResponse
	if err := sendReq(url, winReq, &winRes); err != nil {
	}
	ch <- winRes
}

// sendReq jsonで送信し、規定の型に入れる
func sendReq(url string, sendData interface{}, receiveData interface{}) error {
	sendDataJSON, err := json.Marshal(&sendData)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(sendDataJSON))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(body, &receiveData); err != nil {
		return err
	}

	return nil
}
