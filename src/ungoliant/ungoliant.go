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
	var thread_ptr = flag.Int("threads", 3, "")
	var timeout_ptr = flag.Int("timeout", 5, "")
	var wordlist_ptr = flag.String("wordlist", "res/dirb.txt", "")
	var cx_ptr = flag.String("cse-key", "", "")
	var cse_ptr = flag.String("cx-key", "", "")
	flag.Parse()
	threads := *thread_ptr
	timeout := *timeout_ptr
	wordlist_path := *wordlist_ptr
	cx_key := *cx_ptr
	cse_key := *cse_ptr
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
	_,err = basic_request(proxy_url, timeout, false)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to validate the proxy: %s:%d. Requests will not be recorded!\n", proxy_host, proxy_port)
		proxy = false
	}
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
	//Canary checks to ensure we can tell the difference between FOUND and NOT_FOUND pages.
	fmt.Println("[+] Testing NOT_FOUND detection...")
	checked_hosts := []Host{}
	for index,host := range hosts {
		known_good, canary_urls, err := canary_check(proxy, proxy_host, proxy_port, timeout, threads, host)
		if err == nil {
			hosts[index].heuristic = generate_heuristic(known_good, canary_urls)
			if hosts[index].heuristic.check() {
				checked_hosts = append(checked_hosts, hosts[index])
			}
		}
	}
	fmt.Println("[!] Of the original " + strconv.Itoa(len(hosts)) + " hosts, identified consistent NOT_FOUND heuristics for " + strconv.Itoa(len(checked_hosts)) + " of them.")
	//If configured, use Google CSE to add some candidates to each host.
	if (cx_key != "") && (cse_key != "") {
		fmt.Println("[+] Providing Google CSE checks with the provided CSE and CX keys...")
		for index,host := range checked_hosts {
			retrieved_urls,err := cse_search(cx_key, cse_key, "site:" + host.fqdn)
			if err == nil {
				for _,retrieved_url := range retrieved_urls {
					checked_hosts[index].add_url(retrieved_url)
				}
			}
		}
	}
	//Use the wordlist to generate candidates for each host, and begin bruteforcing.
	for index,host := range checked_hosts {
		fmt.Println("[+] Performing directory bruteforcing on target " + strconv.Itoa(index+1) + " of " + strconv.Itoa(len(checked_hosts)) + ".")
		checked_hosts[index].urls = generate_urls(host, wordlist)
		checked_hosts[index].urls = bruteforce(proxy, proxy_host, proxy_port, timeout, threads, checked_hosts[index].urls)
		checked_hosts[index].flush_urls()
	}
	//Write results to a file.
	err = hosts_to_csv("results.csv", checked_hosts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to write final results to file: results.csv\n")
	} else {
		fmt.Println("[*] Wrote the webserver results to file: results.csv")
	}
	//Done!
	fmt.Println("[+] Finished! Now you can spider and scan the results through Burp/ZAP.")
}

func usage() {
	fmt.Fprintf(os.Stderr, "USAGE: %s <nmap xml file> <proxy IP> <proxy port>\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Optional Flags:\n")
	fmt.Fprintf(os.Stderr, "\t--threads <num>\t\tThe number of threads to use when spidering. [DEFAULT: 3]\n")
	fmt.Fprintf(os.Stderr, "\t--timeout <secs>\tThe timeout value (in seconds) for each request. [DEFAULT: 5]\n")
	fmt.Fprintf(os.Stderr, "\t--wordlist <file>\tA path to a wordlist file for directory bruteforcing. [DEFAULT: \"res/dirb.txt\"]\n")
	fmt.Fprintf(os.Stderr, "\t--cx-key <key>\t\tA Google Custom Search Engine CX key.\n")
	fmt.Fprintf(os.Stderr, "\t--cse-key <key>\t\tA Google Custom Search Engine API key.\n")
	fmt.Fprintf(os.Stderr, "\nExample: %s -t 10 nmap_results.xml 127.0.0.1 8080\n", os.Args[0])
}
