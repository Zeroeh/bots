package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var (
	userAgents = []string{
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:61.0) Gecko/20100101 Firefox/61.0",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/64.0.3282.119 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_7_5) AppleWebKit/536.25 (KHTML, like Gecko) Version/6.0 Safari/536.25",
	}
	//todo: maybe grab this list dynamically?
	serverList = map[string]string{
		"USEast":        "52.23.232.42",
		"USEast2":       "3.88.196.105",
		"USEast3":       "54.157.6.58",
		"USWest":        "13.57.254.131",
		"USWest2":       "54.215.251.128",
		"USWest3":       "54.67.119.179",
		"USSouth":       "52.91.68.60",
		"USSouth2":      "54.183.236.213",
		"USSouth3":      "13.57.182.96",
		"USMidWest":     "13.59.49.120",
		"USMidWest2":    "18.218.255.91", //18.218.255.91
		"USSouthWest":   "54.183.179.205",
		"USNorthWest":   "54.237.156.49",
		"EUEast":        "18.195.167.79",
		"EUWest":        "35.180.229.69",
		"EUWest2":       "34.243.37.98",
		"EUNorth":       "3.122.225.92",
		"EUNorth2":      "52.59.198.155",
		"EUSouth":       "35.180.134.209",
		"EUSouthWest":   "52.47.178.13",
		"AsiaSouthEast": "52.77.221.237",
		"AsiaEast":      "54.199.197.208",
		"Australia":     "54.252.165.65",
	}
)

func serverNameToIP(s string) string {
	//todo?: check if returned string is empty, return default server if so
	return serverList[s]
}

func isIPAddress(s string) bool {
	return strings.Contains(s, ".")
}

func getRandomHeader() string {
	return userAgents[rand.Int31n(int32(len(userAgents)))]
}

func isEmail(email string) bool {
	return strings.Contains(email, "@")
}

func GetTime() int32 {
	return int32(time.Duration(time.Since(time.Now()) / time.Millisecond))
}

func GetIntInRange(min int, max int) int {
	item := int(rand.Int31n(int32(max)))
	if item > min && item < max {
		return item
	} else {
		return GetIntInRange(min, max)
	}
}

func blanketOpenFile(s string, append bool) (*os.File, error) {
	if append == true {
		return os.OpenFile(s, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
	} else {
		return os.OpenFile(s, os.O_CREATE|os.O_RDWR, 0666)
	}
}

func onlyContains(haystack string, needle string) bool {
	if len(needle) == 1 { //confirm we're searching for 1 character
		if len(haystack) == 1 { //no point in checking if it's just one char...
			return false
		}
		haystackLen := len(haystack)
		counts := 0
		for _, v := range haystack {
			if v == rune(needle[0]) { //check each character and see if it matches
				counts++
			}
		}
		if counts == haystackLen {
			return true
		} //if it doesnt match, it has more than 1 character
	} else {
		if haystack == needle {
			return true
		}
	}
	return false
}

func parseNewLines(f *os.File) []string {
	list := make([]string, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		list = append(list, scanner.Text())
	}
	return list
}

func toLower(s string) string {
	return strings.ToLower(s)
}

func stripNewLine(s string) string {
	return strings.ReplaceAll(s, "\n", "")
}

func logln(v ...interface{}) {
	log.Println(v)
}

const (
	Normal  = "\033[0m"
	Black   = "\033[30m"
	Red     = "\033[31m" //previous: \033[1;31m
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	White   = "\033[37m"
)

//ResetColor resets any colors and uses the default color
func ResetColor() {
	fmt.Printf(Normal)
}

//SwitchColor switches the active color to the specified
func SwitchColor(s string) {
	fmt.Printf(s)
}

//Println functions the same as fmt.Println except with an extra color variable
func Println(c string, a ...interface{}) {
	if len(a) == 0 {
		fmt.Println(c, a, Normal)
	} else {
		fmt.Println(c, a, Normal)
	}
}

//Printerr prints the given error in red text
func Printerr(err error) {
	fmt.Printf(Red+"%s\n"+Normal, err.Error())
}

func (a *Account) getURL(s string) (string, error) {
	//make our request appear "legitimate"
	req, err := http.NewRequest(http.MethodGet, s, nil)
	if err != nil {
		log.Println("getURL:", err)
	}
	req.Header.Set("User-Agent", getRandomHeader())
	if a.UseHTTP == true {
		var err error
		proxyURL, err := url.Parse(a.HTTPProxy)
		if err != nil {
			return "", err
		}
		myClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
		myClient.Timeout = time.Second * 5
		// myClient := &http.Client{}
		// trans := &http.Transport{}
		// trans.Proxy = http.ProxyURL(proxyURL)
		// myClient.Transport = trans
		ex, err := myClient.Do(req)
		if err != nil {
			return "", err
		}
		body, err := ioutil.ReadAll(ex.Body) //don't think we'll error if we got to here
		if err != nil {
			return "", err
		}
		ex.Body.Close()
		return string(body), nil
	}
	//no proxy, use actual ip
	myClient := &http.Client{}
	myClient.Timeout = time.Second * 5
	ex, err := myClient.Do(req)
	if err != nil {
		return "", err
	}
	body, err := ioutil.ReadAll(ex.Body)
	if err != nil {
		return "", err
	}
	ex.Body.Close()
	return string(body), nil
}

func quit() {
	os.Exit(0)
}
