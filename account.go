package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//Account is imported from json file
type Account struct {
	Email        string `json:"email"`
	Password     string `json:"password"`
	ServerIP     string `json:"server_ip"`
	FetchNewData bool   `json:"fetch_new_data"`
	CharID       int    `json:"char_id"`
	Module       string `json:"module"`
	UseSocks     bool   `json:"use_socks"`
	SockProxy    string `json:"socks_proxy"`
	UseHTTP      bool   `json:"use_http"`
	HTTPProxy    string `json:"http_proxy"`
}

//GetNewCharID will get a the latest (highest number) char id from char/list. http proxy usage is based on "UseHTTP" setting
//since i couldn't get the xml parser to properly parse the xml from the site i'll have to use old fashioned indexing
func (a *Account) GetNewCharID() int {
	fullURL := "https://realmofthemadgodhrd.appspot.com/char/list?guid=" + a.Email + "&password=" + a.Password
	request, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		fmt.Println("Error forming request:", err)
	}
	var body []byte
	time.Sleep(1500 * time.Millisecond)
	if a.UseHTTP == true {
		proxyURL, _ := url.Parse(a.HTTPProxy)
		myClient := &http.Client{
			Transport: &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			},
		}
		ex, err := myClient.Do(request)
		if err != nil {
			log.Println("GetCharID(Proxy):", err)
			return 0 //if we cant make the request were gonna crash so just return
		}
		body, err = ioutil.ReadAll(ex.Body)
		if err != nil {
			log.Println("GetCharID(Proxy):", err)
		}
		ex.Body.Close()
	} else {
		client := &http.Client{}
		resp, err := client.Do(request)
		if err != nil {
			log.Println("GetCharID:", err)
			return 0
		}
		body, err = ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("GetCharID:", err)
		}
		resp.Body.Close()
	}
	data := string(body)
	//fmt.Println(data)
	//now make sure we aren't in account in use or banned
	if c1 := strings.Contains(data, "Account in use"); c1 == true {
		fmt.Println("Account in use:", a.Email)
		return -1
	}
	if c2 := strings.Contains(data, "wait"); c2 == true {
		fmt.Println("Limited:", a.Email)
		fmt.Println(data)
		return -1
	}
	if c3 := strings.Contains(data, "maintenance"); c3 == true {
		fmt.Println("Banned:", a.Email)
		return -1
		//todo: remove the account from the account list, or write it to a "ban list" file
	}
	//now we can check for chars
	//assuming for now that any bot being run will only have 1 active char at any given time
	ok := strings.Contains(data, "<Char id=")
	if ok == false {
		return 0 //there is no char so we will send create instead of load
	}
	index := strings.Index(data, "<Char id=")
	index2 := strings.Index(data, "<ObjectType>")
	if index2-index < 0 {
		fmt.Println("Error indexing char id")
	}
	stringy1 := data[index:index2]
	charIDSize := len(stringy1) //get the number of digits in charid
	//13 = single digit | 14 = 2 digits | 15 = ....
	var stringy2 string
	switch charIDSize {
	case 13: //ones
		stringy2 = stringy1[charIDSize-3 : charIDSize-2]
	case 14: //tens
		stringy2 = stringy1[charIDSize-4 : charIDSize-2]
	case 15: //hundreds
		stringy2 = stringy1[charIDSize-5 : charIDSize-2]
	case 16: //thousands
		stringy2 = stringy1[charIDSize-6 : charIDSize-2]
	case 17: //ten thousands lol
		stringy2 = stringy1[charIDSize-7 : charIDSize-2]
	default:
		fmt.Println("Unknown char id size:", charIDSize)
	}
	myID, _ := strconv.Atoi(stringy2)
	a.FetchNewData = false
	return myID
	/*charList := Chars{}
	if err := xml.NewDecoder(resp.Body).Decode(&charList); err != nil {
		fmt.Printf("error decoding char/list: %s\n", err)
	}
	fmt.Println(charList)*/
}

//gets the charlist but does nothing with returned data
func (a *Account)getCharList() {
	s := fmt.Sprintf("https://realmofthemadgodhrd.appspot.com/char/list?guid=%s&password=%s&muleDump=true", a.Email, a.Password)
	a.getURL(s)
}
