package main

import "strings"
import "io/ioutil"
import "net/http"
import "net/url"
import "golang.org/x/net/html"

//Here is some code to extract the title from HTML

func isTitleElement(n *html.Node) bool {
	return n.Type == html.ElementNode && n.Data == "title"
}

func traverse(n *html.Node) (string, bool) {
	if isTitleElement(n) {
		if n.FirstChild != nil {
			return n.FirstChild.Data, true
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		result, ok := traverse(c)
		if ok {
			return result, ok
		}
	}

	return "", false
}

func get_html_title(r *http.Response) (string, bool) {
	doc, err := html.Parse(r.Body)
	if err != nil {
		return "",false
	}
	return traverse(doc)
}

//Here is some code to extract the URLs from HTML

func get_html_urls(htmldata string) []string {
	output := []string{}
	z := html.NewTokenizer(strings.NewReader(htmldata))

	for {
	    tt := z.Next()

	    switch {
	    case tt == html.ErrorToken:
	    	// End of the document, we're done
		return output
	    case tt == html.StartTagToken:
		t := z.Token()

		isAnchor := t.Data == "a"
		if isAnchor {
		    for _, a := range t.Attr {
				if a.Key == "href" {
					output = append(output, a.Val)
				}
			}
		}
	    }
	}
}

func get_fqdn_urls(dom string, fqdn string) ([]string, error) {
	//automatically parse and return URLs in HTML that match the provided fqdn
	output := []string{}
	urls := get_html_urls(dom)
	for _,link := range urls {
		parsed,err := url.Parse(link)
		if err == nil && len(link) > 0 {
			if (parsed.Hostname() == fqdn) && (link[0] != '#') {
				output = append(output, link)
			}
		}
	}
	return output,nil
}

//Function for actual scraping and link extraction.

func scrape_url(url string, fqdn string, timeout int, https bool) ([]string,[]string,error) {
	//given a URL and an FQDN, scrape absolute and relative links on the page
	res,err := basic_request(url, timeout, https)
	if err != nil { return []string{},[]string{},err }
	defer res.Body.Close()
	dom,err := ioutil.ReadAll(res.Body)
	if err != nil { return []string{},[]string{},err }
	absolute_urls,err := get_fqdn_urls(string(dom), fqdn)
	if err != nil { return []string{},[]string{},err }
	relative_urls,err := get_fqdn_urls(string(dom), "")
	if err != nil { return []string{},[]string{},err }
	return absolute_urls,relative_urls,nil
}
