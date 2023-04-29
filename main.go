package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"os"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/fireFly-assignment/utils"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/html"
)

type KeyValue struct {
	Key   string
	Value int
}

const (
	regex          = "^[a-zA-Z]+$"
	bankFileName   = "bank-of-words"
	urlFileName    = "endg-urls"
	methodGet      = "GET"
	tmpFolder      = "/tmp/"
	maxConcurrency = 500
)

var (
	words       sync.Map
	client      *http.Client
	bankOfWords map[string]struct{}
	// wait-group to keep track of worker completion
	wg sync.WaitGroup
	// flags
	reqSleepTime, globalCtxTimeOut time.Duration
	// predefined user-agents
	userAgents = []string{
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3",
		"Mozilla/5.0 (Windows NT 6.1; WOW64; rv:54.0) Gecko/20100101 Firefox/54.0",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.81 Safari/537.36",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/58.0.3029.110 Safari/537.3",
	}
)

func main() {
	// initiate flags
	initFlags()

	// set context timeout
	ctx, cancel := context.WithTimeout(context.Background(), globalCtxTimeOut)
	defer cancel()

	// http transport config
	transport := &http.Transport{
		//MaxIdleConns:       20,
		//IdleConnTimeout:    60 * time.Second,
		//DisableCompression: true,
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// assign transport to the client
	client = &http.Client{
		Transport: transport,
	}

	// channel for max concurrent requests
	concurrency := make(chan struct{}, maxConcurrency)

	// init bank of words - validate every word
	bankOfWords = make(map[string]struct{})

	// init alpha-bet regex
	alphaRegexp := regexp.MustCompile(regex)

	//make tmp dir under PWD
	tmpDir, err := utils.MakeTmpUnderPWD(tmpFolder)

	// parse bank-of-words file according to the rules
	_, err = utils.ReadLinesFromFile(bankFileName, true, bankOfWords, alphaRegexp)
	if err != nil {
		log.Error().Msgf("readFromFile failed: %v", err)
	}

	// parse the endg-urls file
	urls, err := utils.ReadLinesFromFile(urlFileName, false, nil, nil)
	if err != nil {
		log.Error().Msgf("readFromFile failed: %v", err)
	}

	// make sure we execute and wait for all routines
	wg.Add(len(urls))
	// spawn workers to process resources
	for _, resource := range urls {
		// send to concurrency channel before each go-routine
		concurrency <- struct{}{}
		go fetchFromURLAndProcessFile(ctx, resource, &wg, &words, concurrency)
	}
	// wait for workers to finish
	wg.Wait()

	// print the top 10 words
	printTopWords(&words)

	//clean-up
	if err := os.RemoveAll(tmpDir); err != nil {
		log.Error().Msgf("RemoveFile failed. file name: %s %v", tmpDir, err)
	}
}

// initiates program flags
func initFlags() {
	flag.DurationVar(&globalCtxTimeOut, "ctx-duration", time.Minute*15, "The context time out (minutes)")
	flag.DurationVar(&reqSleepTime, "req-duration", time.Second*3, "System sleep between HTTP requests (seconds)")
	flag.Parse()
}

// processResource get a file name (contains article body inside) and counts num of appearances of the valid words
func processResource(resource string, wg *sync.WaitGroup, words *sync.Map) {
	defer func() {
		wg.Done()
	}()

	// open the file
	file, err := os.Open(resource)
	if err != nil {
		log.Error().Msgf("os.Open failed: %v", err)
		wg.Done()
		return
	}
	defer file.Close()

	// scanner to read words from file
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)

	// map to store word counts for this resource
	counts := make(map[string]int)

	// count words in the file
	for scanner.Scan() {
		counts[scanner.Text()]++
	}

	// save the valid word and it's count
	for word, count := range counts {
		_, ok := bankOfWords[word]
		if !ok {
			continue
		}
		// get current count
		current, ok := words.Load(word)
		if !ok {
			// save new entry
			words.Store(word, count)
		} else {
			// already in - save the current + count
			words.Store(word, current.(int)+count)
		}
	}
}

// fetchFromURLAndProcessFile get a URL to fetch, and send it's content for processing
func fetchFromURLAndProcessFile(ctx context.Context, url string, wg *sync.WaitGroup, words *sync.Map, maxConCh <-chan struct{}) {
	defer func() {
		// maxConCH represents the max number of go routines at the moment
		// only when a routine has finished, we create space for the next one if the channel is full
		<-maxConCh
	}()

	// because we spawn a lot of go-routines, we need a sleep in between to mitigate load issues
	time.Sleep(reqSleepTime)
	req, err := http.NewRequestWithContext(ctx, methodGet, url, nil)
	if err != nil {
		log.Error().Msgf("http.NewRequest failed: %v", err)
		wg.Done()
		return
	}

	// send a different user-agent header to avoid access denied
	req.Header.Set("User-Agent", userAgents[rand.Intn(len(userAgents))])
	res, err := client.Do(req)
	if err != nil {
		log.Error().Msgf("http.client.Do failed: %v", err)
		wg.Done()
		return
	}

	// use cookies and session management to simulate a user session
	for _, cookie := range res.Cookies() {
		req.AddCookie(cookie)
	}

	// check response status code
	defer res.Body.Close()
	if res.StatusCode != 200 {
		log.Error().Msgf("%s failed with status %d: %v", url, res.StatusCode, err)
		wg.Done()
		return
	}

	// parse html body
	doc, err := html.Parse(res.Body)
	if err != nil {
		log.Error().Msgf("html.Parse failed: %v", err)
		wg.Done()
		return
	}

	// find the first <div> element with class "article-text"
	articleText := utils.FindArticleText(doc)

	// extract the text content from the <div> element
	text := utils.ExtractText(articleText)

	// generate a random string for the file's name
	fileName := tmpFolder + utils.RandomString(10)

	// write file
	utils.WriteStringToFile(fileName, text)

	// send current file for processing in a separate go routine
	go processResource(fileName, wg, words)
}

// printTopWords prints the top 10 words
func printTopWords(words *sync.Map) {
	var keyValueSlice []KeyValue

	// loop over the words and pass them to a slice for sorting
	words.Range(func(key, value interface{}) bool {
		str := key.(string)
		val := value.(int)
		keyValueSlice = append(keyValueSlice, KeyValue{str, val})
		return true
	})

	// sort the slice
	sort.Slice(keyValueSlice, func(i, j int) bool {
		return keyValueSlice[i].Value > keyValueSlice[j].Value
	})

	for i, word := range keyValueSlice {
		if i == 10 {
			return
		}
		fmt.Println(word)
	}
}
