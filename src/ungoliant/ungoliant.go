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
	flag.Parse()
	threads := *thread_ptr
	timeout := *timeout_ptr
	wordlist_path := *wordlist_ptr
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
	_,err = check_web(proxy_host, proxy_port, timeout, false, false)
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
	http_webresults := check_webservers(parsed_hosts, threads, timeout, false)
	if len(http_webresults) > 0 {
		fmt.Println("[!] Found " + strconv.Itoa(len(http_webresults)) + " HTTP servers.")
	}
	fmt.Println("[+] Checking for HTTPS web servers...")
	https_webresults := check_webservers(parsed_hosts, threads, timeout, true)
	if len(https_webresults) > 0 {
		fmt.Println("[!] Found " + strconv.Itoa(len(https_webresults)) + " HTTPS servers.")
	}
	webresults := append(http_webresults, https_webresults...)
	//Write results to a file.
	webresult_str := webresults_to_csv(webresults)
	fd,err = os.Create("webchecker.csv")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to write webserver results to file: webchecker.csv\n")
	} else {
		fd.WriteString(webresult_str)
		fd.Close()
		fmt.Println("[*] Wrote the webserver results to file: webchecker.csv")
	}
	//Initialise a database of Host objects from the webchecker results.
	hosts := []Host{}
	for _,webresult := range webresults {
		new_host := Host{}
		new_host.init(webresult.fqdn, webresult.port, webresult.https)
		hosts = append(hosts, new_host)
	}
	//Canary checks to ensure we can tell the difference between FOUND and NOT_FOUND pages.
	fmt.Println("[+] Testing NOT_FOUND detection...")
	checked_hosts := []Host{}
	for index,host := range hosts {
		canary_urls := canary_check(proxy, proxy_host, proxy_port, timeout, threads, host)
		hosts[index].heuristic = generate_heuristic(canary_urls)
		if hosts[index].heuristic.check() {
			checked_hosts = append(checked_hosts, hosts[index])
		}
	}
	fmt.Println("[!] Of the original " + strconv.Itoa(len(hosts)) + " hosts, identified consistent NOT_FOUND heuristics for " + strconv.Itoa(len(checked_hosts)) + " of them.")
	//Use wordlist to generate candidates for each host, and begin bruteforcing.
	for index,host := range checked_hosts {
		fmt.Println("[+] Performing directory bruteforcing on target " + strconv.Itoa(index+1) + " of " + strconv.Itoa(len(checked_hosts)) + ".")
		checked_hosts[index] = generate_urls(host, wordlist)
		checked_hosts[index].urls = bruteforce(proxy, proxy_host, proxy_port, timeout, threads, checked_hosts[index].urls)
		checked_hosts[index].flush_urls()
	}
	//Write results to a file.
	hosts_str := hosts_to_csv(checked_hosts)
	fd,err = os.Create("results.csv")
	if err != nil {
		fmt.Fprintf(os.Stderr, "[-] Failed to write final results to file: results.csv\n")
	} else {
		fd.WriteString(hosts_str)
		fd.Close()
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
	fmt.Fprintf(os.Stderr, "\nExample: %s -t 10 nmap_results.xml 127.0.0.1 8080\n", os.Args[0])
}

func webresults_to_csv(results []WebResult) string {
	output := "Host,Port,Protocol,Status Code,Status Text\n"
	for _,result := range results {
		output += result.fqdn + "," + strconv.Itoa(result.port) + ","
		if result.https {
			output += "HTTPS,"
		} else {
			output += "HTTP,"
		}
		output += strconv.Itoa(result.statuscode) + "," + result.statustext + "\n"
	}
	return output
}

func hosts_to_csv(hosts []Host) string {
	output := "Host, Port, Url, Status Code, Status Text\n"
	for _,host := range hosts {
		for _,url := range host.urls {
			output += host.fqdn + "," + strconv.Itoa(host.port) + "," + url.url + "," + strconv.Itoa(url.statuscode) + "," + url.statustext + "\n"
		}
	}
	return output
}

