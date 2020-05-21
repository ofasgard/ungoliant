package main

import "sync"
import "fmt"

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

/* request_worker (proxy bool, proxy_host string, proxy_port int, timeout int, urls []*Url, wg *sync.WaitGroup)
*
* Worker function for retrieving individual URLs.
* Simply give it a list of pointers to Url objects, and it retrieves them.
*/

func request_worker(proxy bool, proxy_host string, proxy_port int, timeout int, urls []*Url, wg *sync.WaitGroup) {
	defer wg.Done()
	for index,_ := range urls {
		urls[index].retrieve(proxy, proxy_host, proxy_port, timeout)
	}
}

/*
* bruteforce_worker (proxy bool, proxy_host string, proxy_port int, timeout int, threads int, jobs chan Host, results chan Host, wg *sync.WaitGroup)
*
* Worker function for directory bruteforcing.
* Takes Hosts as input and retrieves all URLs associated with them.
* The completed Hosts are returned as output.
*/

func bruteforce_worker (proxy bool, proxy_host string, proxy_port int, timeout int, threads int, jobs chan Host, results chan Host, wg *sync.WaitGroup) {
	defer wg.Done()
	var request_wg sync.WaitGroup
	for host := range jobs {
		//create list of URLs that haven't been retrieved
		targets := []*Url{}
		for index,_ := range host.urls {
			if !host.urls[index].retrieved {
				targets = append(targets, &host.urls[index])
			}
		}
		//create job lists
		job_lists := [][]*Url{}
		for len(job_lists) < threads {
			job_lists = append(job_lists, []*Url{})
		}
		//populate job lists with URLs
		i := 0
		for index,_ := range targets {
			job_lists[i] = append(job_lists[i], targets[index])
			i++
			if i == threads {
				i = 0
			}
		}
		//create jobs
		for index,_ := range job_lists {
			request_wg.Add(1)
			go request_worker(false, proxy_host, proxy_port, timeout, job_lists[index], &request_wg)
		}
		//wait for jobs to complete and flush URLs
		request_wg.Wait()
		host.flush_urls()
		//if the proxy is enabled, we need to replay any good URLs through the proxy
		if proxy {
			//create list of URLs that haven't been retrieved through proxy
			targets = []*Url{}
			for index,_ := range host.urls {
				if (host.urls[index].retrieved) && (!host.urls[index].retrieved_proxy) {
					targets = append(targets, &host.urls[index])
				}
			}
			//create job lists
			job_lists = [][]*Url{}
			for len(job_lists) < threads {
				job_lists = append(job_lists, []*Url{})
			}
			//populate job lists with URLs
			i = 0
			for index,_ := range targets {
				job_lists[i] = append(job_lists[i], targets[index])
				i++
				if i == threads {
					i = 0
				}
			}
			//create jobs
			for index,_ := range job_lists {
				request_wg.Add(1)
				go request_worker(true, proxy_host, proxy_port, timeout, job_lists[index], &request_wg)
			}
			//wait for jobs to complete
			request_wg.Wait()
		}
		//return
		fmt.Printf("[!] Finished %s:%d\n", host.fqdn, host.port)
		results <- host
	}
}

/*
* bruteforce(proxy bool, proxy_host string, proxy_port int, timeout int, parallel_hosts int, threads int, hosts []Host) []Host
*
* Worker management function for threaded directory bruteforcing.
* Returns a []Host containing the updated hosts after bruteforcing completes.
*/

func bruteforce(proxy bool, proxy_host string, proxy_port int, timeout int, parallel_hosts int, threads int, hosts []Host) []Host {
	output := []Host{}
	//create a number of job lists equal to your host cap
	job_lists := [][]Host{}
	for len(job_lists) < parallel_hosts {
		new_list := []Host {}
		job_lists = append(job_lists, new_list)
	}
	//divide the Hosts equally between the job lists
	index := 0
	for len(hosts) > 0 {
		job_lists[index] = append(job_lists[index], hosts[0])
		hosts = hosts[1:]
		index += 1
		if index == parallel_hosts {
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
		go bruteforce_worker(proxy, proxy_host, proxy_port, timeout, threads, jobs, results, &wg)
		for _,host := range list {
			jobs <- host
		}
		close(jobs)
		result_list = append(result_list, results)
		result_counts = append(result_counts, len(list))
	}
	//wait for all workers to return
	for index,results := range result_list {
		for a:=0; a < result_counts[index]; a++ {
			res := <- results
			output = append(output, res)
		}
	}
	wg.Wait()
	return output
}


