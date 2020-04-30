package main

import "fmt"
import "os"
import "io/ioutil"
import "flag"
import "strconv"
import "strings"

func main() {
	fmt.Println("Then the Unlight of Ungoliant rose up, even to the roots of the trees.")
	//Parse flags and input.
	flag.Usage = usage
	var parallel_ptr = flag.Int("parallel-hosts", 10, "")
	var thread_ptr = flag.Int("threads", 5, "")
	var timeout_ptr = flag.Int("timeout", 5, "")
	var wordlist_ptr = flag.String("wordlist", "res/dirb.txt", "")
	var dork_ptr = flag.Int("dork-depth", 3, "")
	var chrome_ptr = flag.String("chrome-path", "", "")
	flag.Parse()
	parallel_hosts := *parallel_ptr
	threads := *thread_ptr
	timeout := *timeout_ptr
	wordlist_path := *wordlist_ptr
	dork_depth := *dork_ptr
	chrome_path := *chrome_ptr
	//Check we have enough positional arguments.
	if flag.NArg() != 3 {
		usage()
		return
	}
	nmap_path := flag.Arg(0)
	proxy_host := flag.Arg(1)
	proxy_port,err := strconv.Atoi(flag.Arg(2))
	if err != nil {
		//the port could not be converted to an int
		usage()
		return
	}
	//Validate positional arguments.
	if threads < 1 {
		usage()
		return
	}
	if timeout < 1 {
		usage()
		return
	}
	//Check the proxy.
	proxy := true
	proxy_url := "http://" + proxy_host + ":" + strconv.Itoa(proxy_port)
	resp,err := basic_request(proxy_url, timeout, false)
	if err != nil {
		proxy = false
	} else {
		resp.Body.Close()
	}
	//Check for Chrome.
	chrome := check_chrome(chrome_path)
	//Attempt to read and parse the wordlist file.
	fd,err := os.Open(wordlist_path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to open wordlist file for reading: %s\n", wordlist_path)
		return
	}
	defer fd.Close()
	wordlist_data,err := ioutil.ReadAll(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to read data from wordlist file: %s. Try use --wordlist to specify a path.\n", nmap_path)
		return
	}
	wordlist := strings.Split(string(wordlist_data), "\n")
	//Print a summary of configuration info.
	config_summary(proxy, proxy_host, proxy_port, chrome, len(wordlist), parallel_hosts, threads, timeout)
	//Attempt to read and parse the Nmap file.
	fmt.Println("[+] Parsing '" + nmap_path + "'...")
	fd,err = os.Open(nmap_path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to open Nmap file for reading: %s\n", nmap_path)
		return
	}
	defer fd.Close()
	xmlbytes,err := ioutil.ReadAll(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to read XML data from Nmap file: %s\n", nmap_path)
		return
	}
	parsed_hosts,err := parse_nmap(xmlbytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to parse XML from Nmap file: %s\n", nmap_path)
		return
	}
	fmt.Println("[!] Identified " + strconv.Itoa(len(parsed_hosts)) + " open ports.")
	//Now let's check for web servers.
	fmt.Println("[+] Checking for HTTP web servers...")
	http_hosts := checkweb(parsed_hosts, threads, timeout, false)
	if len(http_hosts) > 0 {
		fmt.Println("[!] Found " + strconv.Itoa(len(http_hosts)) + " HTTP servers.")
	}
	fmt.Println("[+] Checking for HTTPS web servers...")
	https_hosts := checkweb(parsed_hosts, threads, timeout, true)
	if len(https_hosts) > 0 {
		fmt.Println("[!] Found " + strconv.Itoa(len(https_hosts)) + " HTTPS servers.")
	}
	hosts := append(http_hosts, https_hosts...)
	//Take a screenshot of each web server if Chrome is installed.
	if chrome != "" {
		fmt.Println("[+] Taking screenshots of identified web servers...")
		create_dir("screenshots")
		for _,host := range hosts {
			filename := fmt.Sprintf("screenshots/%s-port-%d.png", host.fqdn, host.port)
			err := screenshot(host.base_url(), filename, chrome)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[-] Failed to take a screenshot: %s\n", err.Error())
			}
		}
	}
	//Canary checks to ensure we can tell the difference between FOUND and NOT_FOUND pages.
	fmt.Println("[+] Testing NOT_FOUND detection...")
	checked_hosts := []Host{}
	for index,host := range hosts {
		known_good, canary_urls, err := canary_check(false, proxy_host, proxy_port, timeout, host)
		if err == nil {
			hosts[index].heuristic = generate_heuristic(known_good, canary_urls, false)
			if hosts[index].heuristic.check() {
				checked_hosts = append(checked_hosts, hosts[index])
			} else {
				hosts[index].heuristic = generate_heuristic(known_good, canary_urls, true)
				if hosts[index].heuristic.check() { checked_hosts = append(checked_hosts, hosts[index]) }
			}
		}
	}
	fmt.Println("[!] Of the original " + strconv.Itoa(len(hosts)) + " hosts, identified consistent NOT_FOUND heuristics for " + strconv.Itoa(len(checked_hosts)) + " of them.")
	//Use wordlist to generate candidates for each checked host.
	for i,_ := range checked_hosts {
		checked_hosts[i].urls = generate_urls(checked_hosts[i], wordlist)
	}
	//Do some Google dorking to add candidates, if Chrome is installed.
	if chrome != "" {
		fmt.Println("[+] Attempting to Google dork each host...")
		dorked_fqdns := []string{}
		for index,host := range checked_hosts {
			if !string_in_slice(dorked_fqdns, host.fqdn) {
				urls,err := chrome_dork(chrome, host.fqdn, dork_depth)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[-] Google dorking failed on %s: %s\n", host.fqdn, err.Error())
					break
				}
				if len(urls) > 0 {
					fmt.Printf("[!] Retrieved %d URLs for %s.\n", len(urls), host.fqdn)
					for _,retrieved_url := range urls {
						checked_hosts[index].add_url(retrieved_url)
					}
				}
				dorked_fqdns = append(dorked_fqdns, host.fqdn)
			}
		}
	}
	//Begin bruteforcing checked hosts.
	fmt.Println("[+] Performing directory bruteforce on targets...")
	checked_hosts = bruteforce(proxy, proxy_host, proxy_port, timeout, parallel_hosts, threads, checked_hosts)
	fmt.Println("[!] Done.")
	//If Chrome is installed, do some scraping to identify more URLs.
	if chrome != "" {		
		fmt.Println("[+] Attempting to scrape identified pages...")
		scraped_hosts := []Host{}
		for _,host := range checked_hosts {
			new_host := Host{}
			new_host.init(host.fqdn, host.port, host.https)
			new_host.heuristic = host.heuristic
			for _,url := range host.urls {
				absolute_links,relative_links,err := scrape_url(url.url, host.fqdn, chrome)
				if err == nil {
					for _,link := range absolute_links {
						new_host.add_url(link)
					}
					for _,uri := range relative_links {
						link := new_host.base_url() + uri
						new_host.add_url(link)
					}
				}
				
			}
			scraped_hosts = append(scraped_hosts, new_host)
		}
		scraped_hosts = bruteforce(proxy, proxy_host, proxy_port, timeout, parallel_hosts, threads, scraped_hosts)
		checked_hosts = append(scraped_hosts, checked_hosts...)
		fmt.Println("[!] Done.")
	}
	//Write results to a file.
	err = hosts_to_csv("results.csv", checked_hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to write final results to file: results.csv\n")
	} else {
		fmt.Println("[*] Wrote the webserver results to file: results.csv")
	}
	//Done!
	fmt.Println("[+] Finished! Now you can scan the results through Burp/ZAP.")
}

func usage() {
	fmt.Fprintf(os.Stderr, "USAGE: %s <nmap xml file> <proxy IP> <proxy port>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Optional Flags:\n")
	fmt.Fprintf(os.Stderr, "\t--parallel-hosts <num>\tThe maximum number of hosts to scan at once. [DEFAULT: 10]\n")
	fmt.Fprintf(os.Stderr, "\t--threads <num>\t\tThe maximum number of threads to use per host. [DEFAULT: 5]\n")
	fmt.Fprintf(os.Stderr, "\t--timeout <secs>\tThe timeout value (in seconds) for each request. [DEFAULT: 5]\n")
	fmt.Fprintf(os.Stderr, "\t--wordlist <file>\tA path to a wordlist file for directory bruteforcing. [DEFAULT: \"res/dirb.txt\"]\n")
	fmt.Fprintf(os.Stderr, "\t--dork-depth <num>\tHow many pages of Google results to scrape per host (requires Chrome). [DEFAULT: 3]\n")
	fmt.Fprintf(os.Stderr, "\t--chrome-path <path>\tManually specify the location of the Chrome executable (used for screenshots and dorking).\n")
	fmt.Fprintf(os.Stderr, "\nExample: %s -t 10 nmap_results.xml 127.0.0.1 8080\n", os.Args[0])
}

func config_summary(proxy bool, proxy_host string, proxy_port int, chrome string, wordlist_len int, parallel_hosts int, threads int, timeout int) {
	fmt.Fprintf(os.Stdout, "\n")
	if proxy {
		fmt.Fprintf(os.Stdout, "\tProxy: %s:%d\n", proxy_host, proxy_port)
	} else {
		fmt.Fprintf(os.Stdout, "\tProxy: FAILED")
	}
	fmt.Fprintf(os.Stdout, "\tWordlist: %d words\n", wordlist_len)
	fmt.Fprintf(os.Stdout, "\tMax parallel hosts: %d\n", parallel_hosts)
	fmt.Fprintf(os.Stdout, "\tThreads per host: %d\n", threads)
	fmt.Fprintf(os.Stdout, "\tMax possible threads: %d\n", (parallel_hosts * threads))
	fmt.Fprintf(os.Stdout, "\tRequest timeout: %d seconds\n", timeout)
	if chrome == "" {
		fmt.Fprintf(os.Stdout, "\tChrome Path: NOT FOUND\n")
	} else {
		fmt.Fprintf(os.Stdout, "\tChrome Path: %s\n", chrome)
	}
	fmt.Fprintf(os.Stdout, "\n")
}
