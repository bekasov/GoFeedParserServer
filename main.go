package main

import (
	"example.org/uploader"
	"example.org/parser"

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
	//"http://tass.ru/rss/v2.xml",
}


func main() {
	http.HandleFunc("/", httpHandler)
	http.ListenAndServe(":8080", nil)
}

func httpHandler(w http.ResponseWriter, r *http.Request) {
	var inputParams uploader.InputParams = GetInputParams(r.URL.Query())
	response := GetResponse(RssFeeds, inputParams)
	fmt.Fprintf(w, response)
}

func GetResponse(rssFeeds []string, params uploader.InputParams) string {
	var resultChan <-chan parser.ResultData = GetResultChan(rssFeeds, params)

	var responseBuilder strings.Builder
	responseBuilder.WriteString("<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	responseBuilder.WriteString("<rss version=\"2.0\" xmlns:atom=\"http://www.w3.org/2005/Atom\">")
	responseBuilder.WriteString("<channel>")
	responseBuilder.WriteString("<title>Feed parser (search key is " + params.SearchString + ")</title>")
	//responseBuilder.WriteString("<description>Feed parser</description>")
	//responseBuilder.WriteString("<link>http://localhost/rss</link>")

	if params.SortOutput {
		var allFeeds *parser.ResultDataArray = new(parser.ResultDataArray)
		for itemData := range resultChan {
			allFeeds.Add(itemData)
		}
		sort.Sort(allFeeds)
		for _, itemData := range *allFeeds {
			responseBuilder.WriteString(*itemData.Xml)
		}
	} else {
		for itemData := range resultChan {
			responseBuilder.WriteString(*itemData.Xml)
		}
	}

	responseBuilder.WriteString("</channel>")
	responseBuilder.WriteString("</rss>")

	return responseBuilder.String()
}

func GetResultChan(rssFeeds []string, params uploader.InputParams) <-chan parser.ResultData {
	var httpContentChan <-chan uploader.WebPageData = GetHttpContentChan(rssFeeds)
	var result chan parser.ResultData = make(chan parser.ResultData)
	var processWaitHandlers []*sync.WaitGroup
	go func() {
		for webPageData := range httpContentChan {
			var processWaitHandler sync.WaitGroup = sync.WaitGroup{}
			processWaitHandler.Add(1)
			go parser.ParseFeedItems("<url>", webPageData.Content, params, &processWaitHandler, result)
			processWaitHandlers = append(processWaitHandlers, &processWaitHandler)
		}

		go func() {
			for _, processWaitHandlerLoc := range processWaitHandlers {
				processWaitHandlerLoc.Wait()
			}
			close(result)
		}()
	}()

	return result
}

func GetHttpContentChan(rssFeeds []string) <-chan uploader.WebPageData {
	var downloadWaitHandler sync.WaitGroup
	var result = make(chan uploader.WebPageData)
	for _, rssFeed := range rssFeeds {
		downloadWaitHandler.Add(1)
		go uploader.GetHttpContent(rssFeed, &downloadWaitHandler, result)
	}

	go func() {
		downloadWaitHandler.Wait()
		close(result)
	}()

	return result
}

func GetInputParams(getValuesPairs url.Values) uploader.InputParams {
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

	return uploader.InputParams{
		SearchString:  searchString,
		CaseSensitive: caseSensitive,
		SortOutput:    sortOutput,
	}
}
