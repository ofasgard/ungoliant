# ungoliant

A webserver reconnaissance tool that proxies its results through Burp or ZAP.

![example usage](https://user-images.githubusercontent.com/19550999/76216776-18e35200-6209-11ea-93a4-50a2cc3bfb3a.png)

Here's how it works:

1. Use Nmap with the -oX option to scan your targets and output the results to an XML file.
2. Provide the XML file to ungoliant along with the proxy you want to pass requests through.
3. Ungoliant will look for HTTP/HTTPS webservers and enumerate them through the proxy you provided.
4. When it's done, you can go to Burp/ZAP and run spiders or scans on the results.

Ungoliant also logs its results to CSV, so you can parse it yourself if you want.

## Building

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

## TODO

- Parallel host bruteforcing. Why are we only scanning one host at a time?
- Expand screenshot functionality to auto-screenshot based on a keyword, i.e. "finance" in the HTML title or body.
- Implement some actual spidering within the tool before passing it over to Burp/ZAP.

