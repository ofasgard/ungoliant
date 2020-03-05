package main

import "net/http"
import "strconv"

// This struct is used by check_web() to structure the results obtained when attempting to identify webservers.

type WebResult struct {
	fqdn string
	port int
	statuscode int
	statustext string
	https bool
}

// This struct is used to keep track of webservers during the main URL discovery phase.

type Host struct {
	fqdn string
	port int
	https bool
	urls []Url
	heuristic Heuristic
}

func (h *Host) init(fqdn string, port int, https bool) {
	//initialise the host
	h.fqdn = fqdn
	h.port = port
	h.https = https
	h.urls = []Url{}
	h.heuristic = Heuristic{}

}

func (h Host) base_url() string {
	//calculate and return the base URL for this target
	baseurl := ""
	if h.https { baseurl = "https://" } else { baseurl = "http://" }
	baseurl += h.fqdn + ":" + strconv.Itoa(h.port)
	return baseurl
}

func (h *Host) add_url(url string) {
	//add an unretrieved URL to the host
	new_url := Url{}
	new_url.init(url, h.https)
	h.urls = append(h.urls, new_url)
}

func (h *Host) flush_urls() {
	//goes through retrieved URLs, compares them to internal heuristic, discards them any that match
	//used to get rid of NOT_FOUND results from the internal database
	output := []Url{}
	for _,url := range h.urls {
		if url.retrieved{
			if (!h.heuristic.check_url(url)) && (url.statuscode != 0) {
				output = append(output, url)
			}
		} else {
				output = append(output, url)
		}
	}
	h.urls = output
}

//This struct is used to track a Url, whether or not it has been retrieved, and various data like what status code was returned.

type Url struct {
	url string
	https bool
	retrieved bool
	statuscode int
	statustext string
	header_server string
}

func (u *Url) init(url string, https bool) {
	//initialise the URL
	u.url = url
	u.https = https
	u.retrieved = false
}

func (u *Url) retrieve(proxy bool, proxy_host string, proxy_port int, timeout int) error {
	//retrieve a URL via the proxy; this will set the "retrieved" flag regardless of whether it succeeds
	u.retrieved = true
	var resp http.Response
	var err error
	if proxy {
		resp, err = proxy_request(u.url, proxy_host, proxy_port, timeout, u.https)
	} else {
		resp, err = basic_request(u.url, timeout, u.https)
	}
	if err != nil {
		return err
	}
	u.statuscode = resp.StatusCode
	u.statustext = http.StatusText(resp.StatusCode)
	u.header_server = resp.Header.Get("Server")
	return nil
}

// This struct is used to create a heuristic of what a NOT_FOUND response from a webserver looks like. If a field has a nil value, that field can't be used for comparisons.
// If a Url matches the heuristic, it's considered to be NOT_FOUND. If it differs in any valid fields, it's considered to be FOUND.

type Heuristic struct {
	statuscode int
	header_server string
}

func (h Heuristic) check() bool {
	//check if the object contains any valid heuristics
	//if it evaluates to an uninitialised Heuristic object, it contains nothing of value
	if h == (Heuristic{}) {
		return false
	}
	return true
}

func (h Heuristic) check_url(input Url) bool {
	//check if an Url object matches the heuristic
	//returns false as soon as a field that doesn't match is identified; otherwise, returns true
	if (h.statuscode != 0) && (input.statuscode != h.statuscode) {
		return false
	}
	if (h.header_server != "") && (input.header_server != h.header_server) {
		return false
	}
	return true
}
