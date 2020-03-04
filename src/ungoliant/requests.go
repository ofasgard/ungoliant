package main
//Contains functions for checking webservers and performing various web requests.

import "net/http"
import "net/url"
import "crypto/tls"
import "time"
import "strconv"

/*
* check_web(fqdn string, port int, timeout int, use_https bool, verify bool) (WebResult,error)
*
* Check whether a webserver is running on a host and port; if it is, return a WebResult object including status code and text.
* The use_https flag does what it says on the tin. False for HTTP, true for HTTPS.
* The verify flag is used to enable or disable certificate verification. If turned on, invalid TLS certificates will produce an error.
*/

func check_web(fqdn string, port int, timeout int, use_https bool, verify bool) (WebResult,error) {
	output := WebResult{}
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: !verify}}
	client := &http.Client{Transport: tr, Timeout: time.Duration(timeout) * time.Second}
	request_str := ""
	if use_https { request_str += "https://" } else { request_str += "http://" }
	request_str += fqdn + ":" + strconv.Itoa(port) + "/"
	req,err := http.NewRequest("GET", request_str, nil)
	if err != nil {
		return output, err
	}
	resp,err := client.Do(req)
	if err != nil {
		return output, err
	}
	output.fqdn = fqdn
	output.port = port
	output.statuscode = resp.StatusCode
	output.statustext = http.StatusText(resp.StatusCode)
	output.https = use_https
	return output, nil
}

/*
* check_webservers(hosts []Host, threadmax int, timeout int, use_https bool) []WebResult
* 
* Wraps the functionality in check_web() and uses goroutines for threading.
* Returns a slice of WebResults representing the webservers that returned a valid (non-error) response.
*/

func check_webservers(hosts []Host, threadmax int, timeout int, use_https bool) []WebResult {
	output := []WebResult{}
	threads_todo := len(hosts)
	thread_channels := []chan WebResult{}
	//begin queuing up threads until there are none left to do
	for threads_todo > 0 {
		//if we reached our limit, wait for a thread to finish before continuing
		if len(thread_channels) == threadmax {
			result := <-thread_channels[0]
			if result != (WebResult{}) {
				output = append(output, result)
			}
			thread_channels = thread_channels[1:]
		}
		//start a new thread
		next_host := hosts[0]
		hosts = hosts[1:]
		sig := make(chan WebResult, 0)
		go func(host Host, signal chan WebResult) {
			res,_ := check_web(host.fqdn, host.port, timeout, use_https, false)
			signal <-res
		}(next_host,sig)
		thread_channels = append(thread_channels, sig)
		threads_todo -= 1
	}
	//now wait for any outstanding threads to finish
	for len(thread_channels) > 0 {
		result := <-thread_channels[0]
		if result != (WebResult{}) {
			output = append(output, result)
		}
		thread_channels = thread_channels[1:]
	}
	return output
}

/*
* basic_request(request_url string, timeout int, use_https bool) (http.Response,error)
*
* Wrapper for a basic HTTP GET request. Returns a response and error value.
*/

func basic_request(request_url string, timeout int, use_https bool) (http.Response,error) {
	//prepare client and request
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	client := &http.Client{Transport: tr, Timeout: time.Duration(timeout) * time.Second}
	//perform request and return error
	req,err := http.NewRequest("GET", request_url, nil)
	if err != nil {
		return http.Response{},err
	}
	resp,err := client.Do(req)
	if err != nil {
		return http.Response{},err
	}
	return *resp,nil
}

/*
* proxy_request(request_url string, proxy_host string, proxy_port int, timeout int, use_https bool) (http.Response,error)
*
* Make a request through the specified HTTP proxy. Returns a response and error value. Fails if the proxy is down.
*/

func proxy_request(request_url string, proxy_host string, proxy_port int, timeout int, use_https bool) (http.Response,error) {
	//prepare proxy URL
	proxy_str := "http://" + proxy_host + ":" + strconv.Itoa(proxy_port)
	proxy_url,err := url.Parse(proxy_str)
	if err != nil {
		return http.Response{},err
	}
	//prepare client and request
	tr := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, Proxy: http.ProxyURL(proxy_url)}
	client := &http.Client{Transport: tr, Timeout: time.Duration(timeout) * time.Second}
	//perform request and return error
	req,err := http.NewRequest("GET", request_url, nil)
	if err != nil {
		return http.Response{},err
	}
	resp,err := client.Do(req)
	if err != nil {
		return http.Response{},err
	}
	return *resp,nil
}
