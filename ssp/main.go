package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
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

type dspInfo struct {
	bitURL string
	winURL string
}

// SSP メインロジックの構造体
type SSP struct {
	hosts   map[string]dspInfo
	auction []bitResponse
	winReq  winRequest
	adRes   adResponse
}

func main() {
	addr := ":8000"
	s, err := NewSSP()
	if err != nil {
	}
	http.HandleFunc("/", s.AdHandler)
	log.Printf("SSP Server Listening on " + addr + " ...")
	if err := http.ListenAndServe(addr, nil); err != nil {
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

// NewSSP SSPを作成
func NewSSP() (*SSP, error) {
	s := &SSP{}
	if err := s.loadHosts(); err != nil {
		return s, err
	}
	return s, nil
}

// run SSPメインロジック
func (s *SSP) run(adReq adRequest) (adResponse, error) {
	// 1. DSPにgoroutineでリクエストを送る
	if err := s.runBit(adReq.AppID); err != nil {
	}

	fmt.Printf("(%%#v) %#v\n", s.auction)

	// 2. オークションを開く
	if err := s.runAuction(); err != nil {
	}

	// 3. Win通知を送り、広告URLを受け取る
	if err := s.runWin(); err != nil {
	}

	return s.adRes, nil
}

func (s *SSP) loadHosts() error {
	baseURL := "http://localhost:8080"
	dsp := dspInfo{
		bitURL: baseURL + "/",
		winURL: baseURL + "/win",
	}
	s.hosts = map[string]dspInfo{
		"1": dsp,
		"2": dsp,
		"3": dsp,
	}
	return nil
}

// runBit SSPがDSPに並列にbitをリクエストする
func (s *SSP) runBit(AppID string) error {
	var err error
	bitCh := make(chan bitResponse, len(s.hosts))

	eg, ctx := errgroup.WithContext(context.Background())
	ctx, cancel := context.WithTimeout(ctx, 2*1000*time.Millisecond)
	defer cancel()

	for DSPID, host := range s.hosts {
		h := host
		bitReq := &bitRequest{
			AppID: AppID,
			DSPID: DSPID,
		}
		requestFunc := func() error {
			bitRes, err := s.sendBit(ctx, bitCh, h.bitURL, bitReq)
			if err != nil {
				return err
			}
			bitCh <- bitRes
			return nil
		}
		eg.Go(requestFunc)
	}

	if errLocal := eg.Wait(); errLocal != nil {
		err = multierr.Append(err, errLocal)
	}
	if err != nil {
		// ログ
	}
	close(bitCh)

	for bitRes := range bitCh {
		s.auction = append(s.auction, bitRes)
	}

	return nil
}

// runAuction セカンドプライスオークションをする
func (s *SSP) runAuction() error {
	var firstPrice int
	var secondPrice int
	var winDSP bitResponse

	for _, bitRes := range s.auction {
		if firstPrice <= bitRes.Price {
			winDSP = bitRes
			firstPrice = bitRes.Price
		}
		if secondPrice <= bitRes.Price && !(firstPrice == secondPrice) {
			secondPrice = bitRes.Price
		}
	}

	// Todo: auctionがない(DSPがひとつもない)場合
	// -> 自社の広告をセット

	s.winReq = winRequest{
		DSPID: winDSP.DSPID,
		Price: secondPrice,
	}

	return nil
}

// runWin Win通知を送り、広告URLを受け取る
func (s *SSP) runWin() error {
	winCh := make(chan winResponse, 0)
	go s.sendWin(winCh, s.hosts[s.winReq.DSPID].winURL, &s.winReq)
	winRes, ok := <-winCh
	if !ok {
	}
	close(winCh)

	s.adRes = adResponse{
		URL: winRes.URL,
	}

	return nil
}

// sendBit DSPに対してbitリクエストを送る
func (s *SSP) sendBit(ctx context.Context, ch chan bitResponse, url string, bitReq *bitRequest) (bitResponse, error) {
	// bitrequestをjsonにしてhostに送る
	var bitRes bitResponse
	if err := sendReq(ctx, url, bitReq, &bitRes); err != nil {
		return bitRes, err
	}
	return bitRes, nil
}

// sendWin DSPに対してwinリクエストを送る
func (s *SSP) sendWin(ch chan winResponse, url string, winReq *winRequest) {
	var winRes winResponse
	ctx, cancel := context.WithTimeout(context.Background(), 2*1000*time.Millisecond)
	defer cancel()
	if err := sendReq(ctx, url, winReq, &winRes); err != nil {
	}
	ch <- winRes
}

// sendReq jsonで送信し、規定の型に入れる
func sendReq(ctx context.Context, url string, sendData interface{}, receiveData interface{}) error {
	sendDataJSON, err := json.Marshal(&sendData)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(ctx, 1*1000*time.Millisecond)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(sendDataJSON))
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
