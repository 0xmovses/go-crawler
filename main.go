package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

var userAgents = []string{
	"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/39.0.2171.95 Safari/537.36 OPR/26.0.1656.60 (Edition Next)",
	"Opera/8.0 (Windows NT 5.1; U; en)",
	"Mozilla/5.0 (Windows NT 5.1; U; en; rv:1.8.1) Gecko/20061208 Firefox/2.0.0 Opera 9.50",
	"Mozilla/4.0 (compatible; MSIE 6.0; Windows NT 5.1; en) Opera 9.50",
	"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:34.0) Gecko/20100101 Firefox/34.0",
}

func randomUserAgent() string {
	rand.Seed(time.Now().Unix())
	randNum := rand.Int() % len(userAgents)
	return userAgents[randNum]
}

func discoverLinks(response *http.Response, baseURL string) []string {
	if response != nil {
		doc, _ := goquery.NewDocumentFromResponse(response)
		foundUrls := []string{}
		if doc != nil {
			doc.Find("a").Each(func(i int, s *goquery.Selection) {
				res, _ := s.Attr("href")
				foundUrls = append(foundUrls, res)
			})

		}
		return foundUrls
	}
	return []string{}
}

func getRequest(targetURL string) (*http.Response, error) {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", targetURL, nil)
	req.Header.Set("User-Agent", randomUserAgent())

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func checkRelative(href string, baseURL string) string {
	if strings.HasPrefix(href, "/") {
		return fmt.Sprintf("%s%s", baseURL, href)
	}
	return href
}

func resolveRelativeLinks(href string, baseURL string) (bool, string) {
	resultHref := checkRelative(href, baseURL)
	baseParse, _ := url.Parse(baseURL)
	resultParse, _ := url.Parse(resultHref)
	if baseParse != nil && resultParse != nil {
		if baseParse.Host == resultParse.Host {
			return true, resultHref
		}
		return false, ""
	}
	return false, ""
}

var tokens = make(chan struct{}, 5)

func crawl(targetURL string, baseURL string) []string {
	fmt.Println(targetURL)
	tokens <- struct{}{}
	res, _ := getRequest(targetURL)
	<-tokens
	links := discoverLinks(res, baseURL)
	foundUrls := []string{}

	for _, link := range links {
		ok, correctLink := resolveRelativeLinks(link, baseURL)
		if ok {
			if correctLink != "" {
				foundUrls = append(foundUrls, correctLink)
			}
		}
	}
	return foundUrls
}

func main() {
	worklist := make(chan []string)
	n := 1
	baseDomain := "https://www.theguardian.com"
	go func() { worklist <- []string{"https://www.theguardian.com"} }()

	seen := make(map[string]bool)

	for ; n > 0; n-- {
		list := <-worklist

		for _, link := range list {
			if !seen[link] {
				seen[link] = true
				n++

				go func(link string, baseDomain string) {
					foundLinks := crawl(link, baseDomain)
					if foundLinks != nil {
						worklist <- foundLinks
					}
				}(link, baseDomain)
			}
		}
	}
}
