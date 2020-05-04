# ungoliant

A web server reconnaissance tool designed to enumerate entire CIDR ranges at once, and proxy the results through Burp or ZAP.

![example usage](https://user-images.githubusercontent.com/19550999/76216776-18e35200-6209-11ea-93a4-50a2cc3bfb3a.png)

Here's how it works:

1. Use Nmap with the -oX option to scan your targets and output the results to an XML file. Use the -n switch if you don't want to scan hostnames.
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

