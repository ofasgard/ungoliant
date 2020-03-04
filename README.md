# ungoliant

A webserver reconnaissance tool that proxies its results through Burp or ZAP.

![example usage](https://user-images.githubusercontent.com/19550999/75889472-e3afbc00-5e24-11ea-9d61-b8db8b8f5add.png)

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

- Replace the quick and dirty CSV output with the actual CSV library.
- Make it possible to simply run the tool without passing it through a proxy.
- Add the Google CSE functionality (currently implemented but not used).
- Add more heuristics for NOT_FOUND detection, particularly based on HTTP response headers.
- Implement some actual spidering within the tool before passing it over to Burp/ZAP.
- Try to strip out servers that resolve to the same IP and have the same content. (?)

