# ungoliant

Ungoliant is a webserver reconnaissance tool designed to work with an attack proxy, such as Burp or ZAP. With ungoliant, you can enumerate hundreds or even thousands of webservers simultaneously to get a birds' eye view of a target's entire web application estate.

### Features

- Supply targets via Nmap XML or CSV file.
- Automatic webserver identification.
- Screenshots & Google dorking using Chrome.
- Threaded bruteforcing & scraping across many hosts simultaneously.
- Configurable thread limit & timeouts; capable of making 300+ HTTP requests/second.
- Results are sent to Burp/ZAP and written to a CSV file.

## Installation

Ungoliant has the following dependencies:

- golang.org/x/net/html

To fetch the dependencies and build the program, just do:

```shell
$ git clone https://github.com/ofasgard/ungoliant
$ cd ungoliant
$ ./build.sh
$ bin/ungoliant --help
```

For some features like screenshots and Google dorking, you'll need to have Google Chrome installed.

## Help
```shell
Ungoliant v1.0.0
USAGE: bin/ungoliant <xml/csv file> <proxy IP> <proxy port>

Optional Flags:
	--threads <num>			The maximum number of threads per host. [DEFAULT: 10]
	--timeout <secs>		The timeout value (in seconds) for each request. [DEFAULT: 10]
	--wordlist <file>		A path to a wordlist file for directory bruteforcing. [DEFAULT: "res/dirb.txt"]
	--dork-depth <num>		How many pages of Google results to scrape per host (requires Chrome). [DEFAULT: 3]
	--chrome-path <path>		Manually specify the location of the Chrome executable (used for screenshots and dorking).

Example: bin/ungoliant --timeout 10 nmap_results.xml 127.0.0.1 8080
```

### Default Output
```shell
$ bin/ungoliant test.csv
Ungoliant v1.0.0

	[Start time] 25 May 2020 10:41:42
	[Proxy] 127.0.0.1:8080
	[Wordlist] 4620 words
	[Max threads] 10
	[Request timeout] 10 seconds
	[Chrome Path] /usr/bin/google-chrome-stable

[+] Parsing 'test.csv'...
[!] Identified 179 open ports.
[+] Checking for HTTP and HTTPS web servers...
[!] Found 162 HTTP(S) servers.
[+] Taking screenshots of identified web servers...
[+] Testing NOT_FOUND detection...
[!] Of the original 162 hosts, identified heuristics for 162 of them.
[+] Attempting to Google dork each host...
[!] Retrieved 40 URLs for target.net.
[!] Retrieved 8 URLs for target2.net.
[+] Performing directory bruteforce on targets...
	...572 completed, 0 errors
	...1358 completed, 0 errors
	...2154 completed, 0 errors
	...2837 completed, 0 errors
	...3537 completed, 0 errors
	...4210 completed, 0 errors

```

### Increasing Limits
```shell
$ bin/ungoliant --threads 100 --timeout 100 test.xml 
```

## License

GNU GPL v3.0 © Callum Murphy
