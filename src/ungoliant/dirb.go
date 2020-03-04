package main

import "sync"

/*
* generate_urls(target Host, wordlist []string) Host
*
* Apply a wordlist to a Host object to populate its URL database with candidates.
* Returns the same Host object, populated.
*/

func generate_urls(target Host, wordlist []string) Host {
	baseurl := target.base_url()
	for _,candidate := range wordlist {
		new_url := baseurl + "/" + candidate
		target.add_url(new_url)
	}
	return target
}

/*
* bruteforce_worker (proxy_host string, proxy_port int, timeout int, jobs chan Url, results chan Url, wg *sync.WaitGroup)
*
* Worker function for directory bruteforcing.
* Takes URLs as input and retrieves them.
* The retrieved URLs are returned as output.
*/

func bruteforce_worker (proxy_host string, proxy_port int, timeout int, jobs chan Url, results chan Url, wg *sync.WaitGroup) {
	defer wg.Done()
	for job := range jobs {
		job.retrieve(proxy_host, proxy_port, timeout)
		results <- job
	}
}

/*
* func bruteforce(proxy_host string, proxy_port int, timeout int, threads int, urls []Url) []Url 
*
* Worker management function for threaded directory bruteforcing.
* Returns a []Url containing the URLs retrieved by the worker processes.
*/

func bruteforce(proxy_host string, proxy_port int, timeout int, threads int, urls []Url) []Url {
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
	for _,list := range job_lists {
		wg.Add(1)
		jobs := make(chan Url, len(list))
		results := make(chan Url, len(list))
		go bruteforce_worker(proxy_host, proxy_port, timeout, jobs, results, &wg)
		for _,url := range list {
			jobs <- url
		}
		close(jobs)
		for a := 1; a <= len(list); a++ {
			res := <- results
			output = append(output, res)
		}
	}
	//wait for all workers to return
	wg.Wait()
	return output
}

/*
* canary_check(proxy_host string, proxy_port int, timeout int, threads int, target Host) []Url
*
* Given a Host object and some request parameters, do a "canary check" on the webserver.
* This means, generate some random URLs and get the responses from them.
* Returns a slice of Url objects.
*/

func canary_check(proxy_host string, proxy_port int, timeout int, threads int, target Host) []Url {
	canary_wordlist := []string{}
	for len(canary_wordlist) < 5 {
		candidate := random_string(10)
		canary_wordlist = append(canary_wordlist, candidate)
	}
	canary_host := generate_urls(target, canary_wordlist)
	canary_host.urls = bruteforce(proxy_host, proxy_port, timeout, threads, canary_host.urls)
	return canary_host.urls
}

/*
* generate_heuristic(canary_urls []Url) Heuristic
* 
* Given the responses from canary_check(), attempt to generate a Heuristic object.
* Checks if various attributes of each Url object are consistent between canary responses.
* The Heuristic returned from this might be "blank" if none were found, so it should be checked with its check() method.
*/

func generate_heuristic(canary_urls []Url) Heuristic {
	output := Heuristic{}
	if len(canary_urls) < 1 {
		return output
	}
	//check for consistent status code
	check_status := canary_urls[0].statuscode
	valid := true
	for _,url := range canary_urls {
		if url.statuscode != canary_urls[0].statuscode {
			valid = false
		}
	}
	if valid {
		output.statuscode = check_status
	}
	return output
}

