package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"
)

var urlCommon string = ""
var called int64
var w = bufio.NewWriter(os.Stdout)
var calling1 Calling
var calling2 Calling
var ad int64
var xid string

const numRequests = 100

type Calling struct {
	// defining struct variables
	Id         string `json:"id"`
	Actives_at int64  `json:"actives_at"`
	Called_at  int64  `json:"called_at"`
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	wait := new(sync.WaitGroup)
	c1 := make(chan string)
	timeActs := make(chan int64)
	signal := make(chan interface{})

	adjust()
	time.Sleep(1000)

	go post(timeActs, c1)
	xid = <-c1

	for i := 0; i < numRequests; i++ {
		wait.Add(1)
		go put(timeActs, signal, wait)
	}
	go sleeping(timeActs, signal)
	wait.Wait()
}

// サーバとコンピュータ間の接続ディレイ
func adjust() {
	var sum int64 = 0
	for i := 0; i <= 1; i++ {
		resp, _ := http.PostForm(urlCommon, url.Values{"nickname": {"dummy"}})
		respBody, _ := ioutil.ReadAll(resp.Body)
		err1 := json.Unmarshal(respBody, &calling2)
		if err1 != nil {
			fmt.Fprintln(w, err1)
		}
		if sum == 0 {
			sum = calling2.Called_at
		} else {
			sum = sum - calling2.Called_at
		}
	}
	ad = sum
}

//ポストで挑戦、スタート
func post(timeAct chan int64, c1 chan string) {
	defer w.Flush()

	resp, err := http.PostForm(urlCommon, url.Values{"nickname": {"testuser"}})

	respBody, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		str := string(respBody)
		fmt.Fprintln(w, str)
	}
	err1 := json.Unmarshal(respBody, &calling1)
	if err1 != nil {
		fmt.Fprintln(w, err1)
	}
	c1 <- calling1.Id
	called = calling1.Called_at
	timeAct <- calling1.Actives_at
}

//反復フット進行
func put(timeActs chan int64, signal chan interface{}, wait *sync.WaitGroup) {
	defer wait.Done()
	defer w.Flush()

	req, err := http.NewRequest(http.MethodPut, urlCommon, nil)
	req.Header.Set("X-Challenge-Id", xid)
	client := &http.Client{}
	//シグナルによる待ち時間制御
	<-signal
	resp, _ := client.Do(req)

	respBody, err := ioutil.ReadAll(resp.Body)
	if err == nil {
		str := string(respBody)
		fmt.Fprintln(w, str)
		if strings.Contains(str, "result") {
			fmt.Println(str)
			os.Exit(1)
		}
	}
	err1 := json.Unmarshal(respBody, &calling1)
	if err1 != nil {
		fmt.Fprintln(w, err1)
	}
	defer resp.Body.Close()
	called = calling1.Called_at
	timeActs <- calling1.Actives_at

}

//補正時間を加えたスリーピング時間
func sleeping(timeActs <-chan int64, signal chan interface{}) {
	for timeAct := range timeActs {
		sleepTime := timeAct - called + ad
		if sleepTime <= 0 {
			signal <- struct{}{}
			continue
		}
		time.Sleep(time.Duration(sleepTime) * time.Millisecond)
		signal <- struct{}{}
	}
}
