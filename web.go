package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"sync"
)

type WebPageData struct {
	Url     string
	Content string
}

func GetHttpContent(url string, resultWaitHandler *sync.WaitGroup, resultChan chan<- WebPageData) {
	defer resultWaitHandler.Done()
	resp, err := http.Get(url)
	if err != nil {
		log.Println(err)
	}
	defer resp.Body.Close()

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println(err)
	}

	var result WebPageData = WebPageData{
		Url:     url,
		Content: string(bytes),
	}
	resultChan <- result
}

