package main

import (
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var RssFeeds = []string {
	"http://static.feed.rbc.ru/rbc/logical/footer/news.rss",
	"https://lenta.ru/rss",
	"http://tass.ru/rss/v2.xml",
}

type InputParams struct {
	SearchString string
	CaseSensitive bool
	SortOutput bool
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	var inputParams InputParams = GetInputParams(r.URL.Query())
	response := GetResponse(RssFeeds, inputParams)
	fmt.Fprintf(w, response)
}

func GetInputParams(getValuesPairs url.Values) InputParams {
	var searchString string = getValuesPairs.Get("search")
	var sortOutput bool
	var sortOutputParsed error
	if sortOutput, sortOutputParsed = strconv.ParseBool(getValuesPairs.Get("sort")); sortOutputParsed != nil {
		sortOutput = true
	}
	caseSensitive, _ := strconv.ParseBool(getValuesPairs.Get("caseSensitive"))

	if !caseSensitive {
		searchString = strings.ToLower(searchString)
	}

	return InputParams{
		SearchString:  searchString,
		CaseSensitive: caseSensitive,
		SortOutput:    sortOutput,
	}
}

func main() {
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":8080", nil)
}

func GetResponse(rssFeeds []string, params InputParams) string {
	var downloadWaitHandler sync.WaitGroup
	var httpDataChan = make(chan WebPageData)
	for _, rssFeed := range rssFeeds {
		downloadWaitHandler.Add(1)
		go GetHttpContent(rssFeed, &downloadWaitHandler, httpDataChan)
	}

	var processResultChan chan ResultData = make(chan ResultData)

	go func() { // !
		downloadWaitHandler.Wait()
		close(httpDataChan)
	}()

	var processWaitHandlers []*sync.WaitGroup
	for webPageData := range httpDataChan {
		var processWaitHandler sync.WaitGroup = *new(sync.WaitGroup)
		processWaitHandler.Add(1)
		go ParseFeedItems(webPageData.Content, params, &processWaitHandler, processResultChan)
		processWaitHandlers = append(processWaitHandlers, &processWaitHandler)
	}

	var response strings.Builder
	response.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	response.WriteString("<rss version=\"2.0\" xmlns:atom=\"http://www.w3.org/2005/Atom\">")
	response.WriteString("<channel>")
	response.WriteString("<title>Feed parser (search key is " + params.SearchString + ")</title>")
	//response.WriteString("<description>Feed parser</description>")
	//response.WriteString("<link>http://localhost/rss</link>")

	go func() { // !
		for _, processWaitHandlerLoc := range processWaitHandlers {
			processWaitHandlerLoc.Wait()
		}
		close(processResultChan)
	}()

	if params.SortOutput {
		var allFeeds *ResultDataArray = new(ResultDataArray)
		for itemData := range processResultChan {
			allFeeds.Add(itemData)
		}
		sort.Sort(allFeeds)
		for _, itemData := range *allFeeds {
			response.WriteString(*itemData.Xml)
		}
	} else {
		for itemData := range processResultChan {
			response.WriteString(*itemData.Xml)
		}
	}

	response.WriteString("</channel>")
	response.WriteString("</rss>")

	return response.String()
}