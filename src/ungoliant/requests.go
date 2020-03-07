package main
//Contains functions for checking webservers and performing various web requests.

import "net/http"
import "net/url"
import "crypto/tls"
import "time"
import "strconv"
import "sync"

// the worker either returns Hosts with correctly initialised HTTPS, or it returns uninitialised Hosts that can be discarded

func checkweb_worker(timeout int, use_https bool, verify bool, jobs chan Host, results chan Host, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		request_str := ""
		if use_https { request_str += "https://" } else { request_str += "http://" }
		request_str += job.fqdn + ":" + strconv.Itoa(job.port) + "/"
		_,err := basic_request(request_str, timeout, use_https)
		if err == nil {
			job.init(job.fqdn, job.port, use_https)
			results <- job
		} else {
			results <- Host{}
		}
	}
}

func checkweb(hosts []Host, threads int, timeout int, use_https bool) []Host {
	output := []Host{}
	//divide the hosts into equally sized job lists
	job_lists := [][]Host{}
	for len(job_lists) < threads {
		new_list := []Host {}
		job_lists = append(job_lists, new_list)
	}
	index := 0
	for len(hosts) > 0 {
		job_lists[index] = append(job_lists[index], hosts[0])
		hosts = hosts[1:]
		index += 1
		if index == threads {
			index = 0
		}
	}
	//assign workers to each job list
	var wg sync.WaitGroup
	result_list := []chan Host{}
	result_counts := []int{}
	for _,list := range job_lists {
		wg.Add(1)
		jobs := make(chan Host, len(list))
		results := make(chan Host, len(list))
		go checkweb_worker(timeout, use_https, false, jobs, results, &wg)
		for _,host := range list {
			jobs <- host
		}
		close(jobs)
		result_list = append(result_list, results)
		result_counts = append(result_counts, len(list))
	}
	//wait for all workers to return
	for index,results := range result_list {
		for a := 0; a < result_counts[index]; a++ {
			res := <- results
			if res.fqdn != "" {
				output = append(output, res)
			}
		}
	}
	wg.Wait()
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
	defer resp.Body.Close()
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
	defer resp.Body.Close()
	return *resp,nil
}

