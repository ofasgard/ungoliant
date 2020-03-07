# ungoliant

A webserver reconnaissance tool that proxies its results through Burp or ZAP.

![example usage](https://user-images.githubusercontent.com/19550999/75923495-1bd2f100-5e5d-11ea-973e-82a628f6971e.png)

Here's how it works:

1. Use Nmap with the -oX option to scan your targets and output the results to an XML file.
2. Provide the XML file to ungoliant along with the proxy you want to pass requests through.
3. Ungoliant will look for HTTP/HTTPS webservers and enumerate them through the proxy you provided.
4. When it's done, you can go to Burp/ZAP and run spiders or scans on the results.

Ungoliant also logs its results to a few CSV files, so you can parse them yourself if you want.

## Building

Ungoliant doesn't have any external dependencies besides Go itself. To build it, just do:

```shell
$ git clone https://github.com/ofasgard/ungoliant
$ cd ungoliant
$ ./build.sh
$ bin/ungoliant --help
```

## TODO

- Make basic_request() and proxy_request() return a pointer to the response, and fix code that uses them so that it properly closes the response body.
- Do more testing on representative examples.
- Add more configuration to web requests, such as a custom User Agent or Authentication headers.
- Implement some actual spidering within the tool before passing it over to Burp/ZAP.
