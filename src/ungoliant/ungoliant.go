package main

import "fmt"
import "os"
import "io/ioutil"
import "flag"
import "strconv"
import "strings"
import "time"
import "math/rand"
import "path/filepath"

func main() {
	fmt.Println("Then the Unlight of Ungoliant rose up, even to the roots of the trees.")
	//Generate seed for random operations (pseudo-randomness is acceptable in this use case).
	rand.Seed(time.Now().UnixNano())
	//Parse flags and input.
	flag.Usage = usage
	var thread_ptr = flag.Int("threads", 10, "")
	var timeout_ptr = flag.Int("timeout", 5, "")
	var wordlist_ptr = flag.String("wordlist", "res/dirb.txt", "")
	var dork_ptr = flag.Int("dork-depth", 3, "")
	var chrome_ptr = flag.String("chrome-path", "", "")
	flag.Parse()
	threads := *thread_ptr
	timeout := *timeout_ptr
	wordlist_path := *wordlist_ptr
	dork_depth := *dork_ptr
	chrome_path := *chrome_ptr
	//Validate positional arguments.
	if flag.NArg() < 1 {
		usage()
		return
	}
	input_path := flag.Arg(0)
	proxy_host := "127.0.0.1"
	proxy_port := 8080
	var err error
	if flag.NArg() >= 2 {
		proxy_host = flag.Arg(1)
	}
	if flag.NArg() >= 3 {
		proxy_port,err = strconv.Atoi(flag.Arg(2))
		if err != nil {
			usage()
			return
		}
	}
	//Validate optional arguments.
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
	resp,err := basic_request(proxy_url, timeout)
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
		fmt.Fprintf(os.Stderr, "[-] Failed to read data from wordlist file: %s. Try use --wordlist to specify a path.\n", wordlist_path)
		return
	}
	wordlist := strings.Split(string(wordlist_data), "\n")
	//Measure start time & print a summary of configuration info.
	start_time := time.Now()
	config_summary(proxy, proxy_host, proxy_port, chrome, len(wordlist), threads, timeout, start_time)
	//Attempt to read and parse the Nmap or CSV file.
	fmt.Println("[+] Parsing '" + input_path + "'...")
	fd,err = os.Open(input_path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to open input file for reading: %s\n", input_path)
		return
	}
	defer fd.Close()
	input_bytes,err := ioutil.ReadAll(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to read data from input file: %s\n", input_path)
		return
	}
	input_ext := strings.ToLower(filepath.Ext(input_path))
	var parsed_hosts []Host
	switch input_ext {
		case ".csv":
			parsed_hosts,err = import_csv(input_bytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[-] Failed to parse target hosts and ports from CSV file: %s\n", input_path)
				return
			}
		case ".xml":
			parsed_hosts,err = import_nmap(input_bytes)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[-] Failed to parse target data from Nmap XML file: %s\n", input_path)
				return
			}
		default:
			fmt.Fprintf(os.Stderr, "[-] Invalid extension: %s - please provide either an Nmap XML file or a CSV file.\n", input_ext)
			return
	}
	fmt.Println("[!] Identified " + strconv.Itoa(len(parsed_hosts)) + " open ports.")
	//Now let's check for web servers.
	fmt.Println("[+] Checking for HTTP and HTTPS web servers...")
	hosts := checkweb(parsed_hosts, threads, timeout)
	if len(hosts) > 0 {
		fmt.Println("[!] Found " + strconv.Itoa(len(hosts)) + " HTTP(S) servers.")
	} else {
		fmt.Fprintf(os.Stderr, "[-] Failed to identify any open HTTP(S) servers to test!\n")
		return
	}
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
	for index,_ := range hosts {
		err = hosts[index].generate_notfound(timeout)
		if err == nil {
			checked_hosts = append(checked_hosts, hosts[index])
		}
	}
	fmt.Println("[!] Of the original " + strconv.Itoa(len(hosts)) + " hosts, identified heuristics for " + strconv.Itoa(len(checked_hosts)) + " of them.")
	//Update thread count to match number of hosts.
	threads = threads * len(checked_hosts)
	if threads < 1 { threads = 1 }
	//Use wordlist to generate candidates for each checked host.
	for i,_ := range checked_hosts {
		checked_hosts[i].generate_urls(wordlist)
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
	checked_hosts = bruteforce(proxy, proxy_host, proxy_port, timeout, threads, checked_hosts)
	//Do some scraping to identify more URLs.
	fmt.Println("[+] Attempting to scrape identified pages...")
	checked_hosts = scrape(checked_hosts, timeout, threads)
	checked_hosts = bruteforce(proxy, proxy_host, proxy_port, timeout, threads, checked_hosts)
	//Write results to a file.
	err = hosts_to_csv("results.csv", checked_hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to write final results to results.csv: %s\n", err.Error())
	} else {
		fmt.Println("[*] Wrote the webserver results to file: results.csv")
	}
	//Done!
	finish_time := time.Now()
	duration := finish_time.Sub(start_time)
	fmt.Fprintf(os.Stdout, "[+] Finished! Total elapsed time: %s\n", duration.Round(time.Second).String())
}

func usage() {
	fmt.Fprintf(os.Stderr, "USAGE: %s <nmap xml file> <proxy IP> <proxy port>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Optional Flags:\n")
	fmt.Fprintf(os.Stderr, "\t--threads <num>\t\tThe maximum number of threads per host. [DEFAULT: 10]\n")
	fmt.Fprintf(os.Stderr, "\t--timeout <secs>\tThe timeout value (in seconds) for each request. [DEFAULT: 5]\n")
	fmt.Fprintf(os.Stderr, "\t--wordlist <file>\tA path to a wordlist file for directory bruteforcing. [DEFAULT: \"res/dirb.txt\"]\n")
	fmt.Fprintf(os.Stderr, "\t--dork-depth <num>\tHow many pages of Google results to scrape per host (requires Chrome). [DEFAULT: 3]\n")
	fmt.Fprintf(os.Stderr, "\t--chrome-path <path>\tManually specify the location of the Chrome executable (used for screenshots and dorking).\n")
	fmt.Fprintf(os.Stderr, "\nExample: %s --timeout 10 nmap_results.xml 127.0.0.1 8080\n", os.Args[0])
}

func config_summary(proxy bool, proxy_host string, proxy_port int, chrome string, wordlist_len int, threads int, timeout int, start_time time.Time) {
	fmt.Fprintf(os.Stdout, "\n")
	fmt.Fprintf(os.Stdout, "\t[Start time] %s\n", start_time.Format("2 Jan 2006 15:04:05"))
	if proxy {
		fmt.Fprintf(os.Stdout, "\t[Proxy] %s:%d\n", proxy_host, proxy_port)
	} else {
		fmt.Fprintf(os.Stdout, "\t[Proxy] FAILED\n")
	}
	fmt.Fprintf(os.Stdout, "\t[Wordlist] %d words\n", wordlist_len)
	fmt.Fprintf(os.Stdout, "\t[Threads per host] %d\n", threads)
	fmt.Fprintf(os.Stdout, "\t[Request timeout] %d seconds\n", timeout)
	if chrome == "" {
		fmt.Fprintf(os.Stdout, "\t[Chrome Path] NOT FOUND\n")
	} else {
		fmt.Fprintf(os.Stdout, "\t[Chrome Path] %s\n", chrome)
	}
	fmt.Fprintf(os.Stdout, "\n")
}
