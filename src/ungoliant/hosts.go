package main

import "net/http"
import "strconv"

// This struct is used to keep track of webservers during the main URL discovery phase.

type Host struct {
	fqdn string
	port int
	https bool
	urls []Url
	notfound []Url
}

func (h *Host) init(fqdn string, port int, https bool) {
	//initialise the host
	h.fqdn = fqdn
	h.port = port
	h.https = https
	h.urls = []Url{}
	h.notfound = []Url{}
	//add base URL to wordlist
	base := Url{}
	base.init(h.base_url(), h.https)
	h.urls = append(h.urls, base)
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
	if h.check_url(url) {
		new_url := Url{}
		new_url.init(url, h.https)
		h.urls = append(h.urls, new_url)
	}
}

func (h Host) check_url(in_url string) bool {
	//check that a URL does not already exist in the Host
	for _,url := range h.urls {
		if url.url == in_url {
			return false
		}
	}
	return true
}

func (h *Host) generate_notfound(timeout int) error {
	//generate the internal heuristic (NOT_FOUND)
	//this is done by generating 3 random URLs and requesting them
	//they are then stored as a benchmark
	for i := 0; i < 3; i++ {
		random_url := h.base_url() + "/" + random_string(10)
		candidate := Url{}
		candidate.init(random_url, h.https)
		candidate.retrieve(false, "", 0, timeout)
		if candidate.err != nil { return candidate.err }
		h.notfound = append(h.notfound, candidate)
	}
	return nil
}

func (h Host) check_notfound(candidate Url) bool {
	//check to see whether a retrieved URL matches the internal heuristic (NOT_FOUND)
	//it only counts as a match if the heuristic is consistent across all 3 NOT_FOUND URLs
	//true = NOT_FOUND, false = FOUND/INTERESTING
	if (candidate.statuscode != h.notfound[0].statuscode) && int_slice_equal(h.notfound[0].statuscode, h.notfound[1].statuscode, h.notfound[2].statuscode) { return false }
	if (candidate.header_server != h.notfound[0].header_server) && string_slice_equal(h.notfound[0].header_server, h.notfound[1].header_server, h.notfound[2].header_server) { return false }
	if (candidate.proto != h.notfound[0].proto) && string_slice_equal(h.notfound[0].proto, h.notfound[1].proto, h.notfound[2].proto) { return false }
	if (candidate.html_title != h.notfound[0].html_title) && string_slice_equal(h.notfound[0].html_title, h.notfound[1].html_title, h.notfound[2].html_title) { return false }
	return true
}

func (h *Host) flush_urls() {
	//goes through retrieved URLs, compares them to internal heuristic, discards any that match
	//used to get rid of NOT_FOUND results from the internal database, and URLs that returned an error
	output := []Url{}
	for _,url := range h.urls {
		if url.retrieved{
			if (!h.check_notfound(url)) && (url.err == nil) {
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
	retrieved_proxy bool
	err error
	statuscode int
	statustext string
	header_server string
	proto string
	html_title string
}

func (u *Url) init(url string, https bool) {
	//initialise the URL
	u.url = url
	u.https = https
	u.retrieved = false
	u.retrieved_proxy = false
	u.err = nil
}

func (u *Url) retrieve(proxy bool, proxy_host string, proxy_port int, timeout int) {
	//retrieve a URL via the proxy; this will set the "retrieved" flag regardless of whether it succeeds
	u.retrieved = true
	if proxy { u.retrieved_proxy = true }
	var resp *http.Response
	var err error
	if proxy {
		resp, err = proxy_request(u.url, proxy_host, proxy_port, timeout)
	} else {
		resp, err = basic_request(u.url, timeout)
	}
	if err != nil {
		u.err = err
		return
	}
	defer resp.Body.Close()
	u.statuscode = resp.StatusCode
	u.statustext = http.StatusText(resp.StatusCode)
	u.header_server = resp.Header.Get("Server")
	u.proto = resp.Proto
	title,success := get_html_title(resp)
	if success == true {
		u.html_title = title
	}
}

