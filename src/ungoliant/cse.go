package main
//Contains functions for dorking using Google Custom Search Engines.

import "net/http"
import "crypto/tls"
import "net/url"
import "time"
import "io/ioutil"
import "encoding/json"

/*
* cse_search(cxkey string, csekey string, query string) ([]string,error)
*
* Perform a single search across a Google Custom Search Engine and retrieve URLs from the response.
* Requires valid credentials for the CSE API: a CX id and an API key.
*/

func cse_search(cxkey string, csekey string, query string) ([]string,error) {
	results := []string{}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: false}}
	client := &http.Client{Transport: tr, Timeout: 5 * time.Second}
	url := "https://www.googleapis.com/customsearch/v1?cx=" + cxkey + "&key=" + csekey + "&q=" + url.QueryEscape(query)
	req,err := http.NewRequest("GET", url, nil)
	if err != nil {
		return results,err
	}
	resp,err := client.Do(req)
	if err != nil {
		return results,err
	}
	body,err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return results,err
	}
	results = cse_parse(body)
	return results,nil
}

/*
* cse_parse(cse_json []byte) []string
*
* A simple helper function to parse the JSON returned by the Google CSE API.
* Returns an array of strings containing the fetched URLs.
*/

func cse_parse(cse_json []byte) []string {
	output := []string{}
	data := CSEData{}
	json.Unmarshal(cse_json, &data)
	for _,item := range data.Items {
		output = append(output, item.FormattedUrl)
	}
	return output
}

// Structs used by encoding/json to parse the CSE JSON.

type CSEData struct {
	Items []CSEItem
}

type CSEItem struct {
	FormattedUrl string
}
