package parser

import (
	"encoding/xml"
	"io"
	"log"
	"strings"
	"sync"

	"example.org/uploader"
)

type XmlFeedItem struct {
	XMLName     xml.Name
	Url         string `xml:"link"`
	Title       string `xml:"title"`
	Description string `xml:"description,omitempty"`
	Guid        string `xml:"guid"`
	Author      string `xml:"author,omitempty"`
	PubDate     string `xml:"pubDate"`
	Category    string `xml:"category,omitempty"`
}

type ResultData struct {
	Xml          *string
	EntriesCount int
}

func ParseFeedItems(url string, content string, params uploader.InputParams, processWaitHandler *sync.WaitGroup, processResultChan chan<- ResultData) {
	defer processWaitHandler.Done()
	var xmlDecoder *xml.Decoder = xml.NewDecoder(strings.NewReader(content))

	for {
		currentToken, tokenErr := xmlDecoder.Token()

		if tokenErr != nil {
			if tokenErr == io.EOF {
				break
			}
			log.Println("Error parsing content from " + url)
			log.Println(content)
			log.Println(tokenErr.Error())
			break
		}
		switch t := currentToken.(type) {
		case xml.StartElement:
			if t.Name.Local == "item" {
				var currentFeedItem XmlFeedItem

				if err := xmlDecoder.DecodeElement(&currentFeedItem, &t); err != nil {
					log.Println(err.Error())
				} else {
					processWaitHandler.Add(1)
					go ProcessFeedItems(currentFeedItem, params, processWaitHandler, processResultChan)
				}
			}
			break
		}
	}
}

func ProcessFeedItems(feedItem XmlFeedItem, params uploader.InputParams, processWaitHandler *sync.WaitGroup, resultChan chan<- ResultData) {
	defer processWaitHandler.Done()

	var stringCompareWaitHandler sync.WaitGroup

	getCount := func(source string, params uploader.InputParams, result *int) {
		defer stringCompareWaitHandler.Done()

		if !params.CaseSensitive {
			source = strings.ToLower(source)
		}
		*result = strings.Count(source, params.SearchString)
	}

	var titleResult, descResult int
	stringCompareWaitHandler.Add(2)
	go getCount(feedItem.Title, params, &titleResult)
	go getCount(feedItem.Description, params, &descResult)
	stringCompareWaitHandler.Wait()

	var searchStringEntries int = titleResult + descResult

	if searchStringEntries > 0 {
		var resultBuilder strings.Builder
		var xmlOutput *xml.Encoder = xml.NewEncoder(&resultBuilder)
		xmlOutput.EncodeElement(feedItem, xml.StartElement{Name: xml.Name{Local: "item"}}) // !!
		xmlOutput.Flush()
		resultXml := resultBuilder.String()

		resultChan <- ResultData{
			Xml:          &resultXml,
			EntriesCount: searchStringEntries,
		}
	}
}

type ResultDataArray []ResultData

func (self ResultDataArray) Len() int {
	return len(self)
}

func (self ResultDataArray) Less(i, j int) bool {
	return self[i].EntriesCount > self[j].EntriesCount
}

func (self ResultDataArray) Swap(i, j int) {
	self[i], self[j] = self[j], self[i]
}

func (self *ResultDataArray) Add(item ResultData) {
	*self = append(*self, item)
}
