package main

import "sync"

/*
* generate_urls(target Host, wordlist []string) Host
*
* Apply a wordlist to a Host object to populate its URL database with candidates.
* Returns the same Host object, populated.
*/

func generate_urls(target Host, wordlist []string) []Url {
	baseurl := target.base_url()
	for _,candidate := range wordlist {
		new_url := baseurl + "/" + candidate
		target.add_url(new_url)
	}
	return target.urls
}

/*
* bruteforce_worker (proxy bool, proxy_host string, proxy_port int, timeout int, jobs chan Url, results chan Url, wg *sync.WaitGroup)
*
* Worker function for directory bruteforcing.
* Takes URLs as input and retrieves them.
* The retrieved URLs are returned as output.
*/

func bruteforce_worker (proxy bool, proxy_host string, proxy_port int, timeout int, jobs chan Url, results chan Url, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		job.retrieve(proxy, proxy_host, proxy_port, timeout)
		results <- job
	}
}

/*
* bruteforce(proxy bool, proxy_host string, proxy_port int, timeout int, threads int, urls []Url) []Url 
*
* Worker management function for threaded directory bruteforcing.
* Returns a []Url containing the URLs retrieved by the worker processes.
*/

func bruteforce(proxy bool, proxy_host string, proxy_port int, timeout int, threads int, urls []Url) []Url {
	output := []Url{}
	//divide the urls up into equally sized job lists
	job_lists := [][]Url{}
	for len(job_lists) < threads {
		new_list := []Url {}
		job_lists = append(job_lists, new_list)
	}
	index := 0
	for len(urls) > 0 {
		job_lists[index] = append(job_lists[index], urls[0])
		urls = urls[1:]
		index += 1
		if index == threads {
			index = 0
		}
	}
	//assign workers to each job list
	var wg sync.WaitGroup
	result_list := []chan Url{}
	result_counts := []int{}
	for _,list := range job_lists {
		wg.Add(1)
		jobs := make(chan Url, len(list))
		results := make(chan Url, len(list))
		go bruteforce_worker(proxy, proxy_host, proxy_port, timeout, jobs, results, &wg)
		for _,url := range list {
			jobs <- url
		}
		close(jobs)
		result_list = append(result_list, results)
		result_counts = append(result_counts, len(list))
	}
	//wait for all workers to return
	for index,results := range result_list {
		for a := 0; a < result_counts[index]; a++ {
			res := <- results
			output = append(output, res)
		}
	}
	wg.Wait()
	return output
}

/*
* canary_check(proxy bool, proxy_host string, proxy_port int, timeout int, threads int, target Host) (Url,[]Url)
*
* Given a Host object and some request parameters, do a "canary check" on the webserver.
* Request the base URL and 5 randomly-generated URLs.
* Returns the retrieved URLs.
*/

func canary_check(proxy bool, proxy_host string, proxy_port int, timeout int, threads int, target Host) (Url,[]Url,error) {
	base_url := Url{}
	base_url.init(target.base_url(), target.https)
	canary_wordlist := []string{}
	for len(canary_wordlist) < 5 {
		candidate := random_string(10)
		canary_wordlist = append(canary_wordlist, candidate)
	}
	target.urls = generate_urls(target, canary_wordlist)
	known_good := bruteforce(proxy, proxy_host, proxy_port, timeout, threads, []Url{base_url})
	canary_urls := bruteforce(proxy, proxy_host, proxy_port, timeout, threads, target.urls[1:])
	return known_good[0], canary_urls, known_good[0].err
}

/*
* generate_heuristic(known_good Url, canary_urls []Url, fuzzy bool) Heuristic
* 
* Given the responses from canary_check(), attempt to generate a Heuristic object.
* Checks if various attributes of each Url object are consistent between canary responses.
* The Heuristic returned from this might be "blank" if none were found, so it should be checked with its check() method.
* If the "fuzzy" flag is set, we don't compare it to "known good" URLs and just look for consistent NOT_FOUND results.
*/

func generate_heuristic(known_good Url, canary_urls []Url, fuzzy bool) Heuristic {
	output := Heuristic{}
	if len(canary_urls) < 1 {
		return output
	}
	//check for consistent status code
	check_status := canary_urls[0].statuscode
	valid := true
	for _,url := range canary_urls {
		if url.statuscode != check_status {
			valid = false
		}
	}
	if valid {
		output.statuscode = check_status
	}
	//check for consistent Server header
	check_server := canary_urls[0].header_server
	valid = true
	for _,url := range canary_urls {
		if url.header_server != check_server {
			valid = false
		}
	}
	if valid {
		output.header_server = check_server
	}
	//check for consistent HTTP version
	check_proto := canary_urls[0].proto
	valid = true
	for _,url := range canary_urls {
		if url.proto != check_proto {
			valid = false
		}
	}
	if valid {
		output.proto = check_proto
	}
	//check for consistent HTML title
	check_html_title := canary_urls[0].html_title
	valid = true	
	for _,url := range canary_urls {
		if url.html_title != check_html_title {
			valid = false
		}
	}
	if valid {
		output.html_title = check_html_title
	}
	//compare to the "known good" URL and discard any results that match
	if !fuzzy {
		if output.statuscode == known_good.statuscode { output.statuscode = 0 }
		if output.header_server == known_good.header_server { output.header_server = "" }
		if output.proto == known_good.proto { output.proto = "" }
		if output.html_title == known_good.html_title { output.html_title = "" }
	}
	//done
	return output
}


