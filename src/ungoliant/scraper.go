package main

import "sync"
import "math/rand"
import "strings"
import "io/ioutil"
import "net/http"
import "net/url"
import "golang.org/x/net/html"

func scrape(targets []Host, timeout int, threads int) []Host {
	sc := Scraper{}
	sc.init(timeout, threads)
	for index,_ := range targets {
		sc.add(&targets[index])
	}
	sc.run()
	for _,result := range sc.results {
		if result.err == nil {
			base := result.parent.base_url()
			for _,absolute_result := range result.absolute_results {
				result.parent.add_url(absolute_result)
			}
			for _,relative_result := range result.relative_results {
				result.parent.add_url(base + relative_result)
			}
		}
	}
	return targets
}

// SCRAPER BEGINS HERE

type Scraper struct {
	timeout int
	threads int

	targets []ScraperTarget
	results []ScraperTarget
	wg sync.WaitGroup
}

type ScraperTarget struct {
	url Url
	parent *Host
	absolute_results []string
	relative_results []string
	err error
}

func (s *Scraper) init(timeout int, threads int) {
	s.timeout = timeout
	s.threads = threads
	
	s.targets = []ScraperTarget{}
	s.results = []ScraperTarget{}
}

func (s *Scraper) add(target *Host) {
	for _,new_url := range target.urls {
		new_target := ScraperTarget{url: new_url, parent: target}
		s.targets = append(s.targets, new_target)
	}
}

func (s *Scraper) run() {
	//Randomise the order of the target list.
	rand.Shuffle(len(s.targets), func(i, j int) { s.targets[i], s.targets[j] = s.targets[j], s.targets[i] })
	//Create a number of worker goroutines equal to the thread limit.
	input_channels := []chan ScraperTarget{}
	output_channels := []chan ScraperTarget{}
	for i := 0; i < s.threads; i++ {
		s.wg.Add(1)
		input_chan := make(chan ScraperTarget)
		output_chan := make(chan ScraperTarget)
		input_channels = append(input_channels, input_chan)
		output_channels = append(output_channels, output_chan)
		go scraper_worker(&s.wg, input_chan, output_chan, s.timeout)
	}
	//Hand out targets to all the goroutines we just created, using the input channels.
	current_channel := 0
	for index,_ := range s.targets {
		input_channels[current_channel] <- s.targets[index]
		current_channel++
		if current_channel == s.threads { current_channel = 0 }
	}
	//Close the input channels.
	for index,_ := range input_channels {
		close(input_channels[index])
	}
	//Get output from all goroutines as they complete.
	for _,output_channel := range output_channels {
		for {
			val,ok := <-output_channel
			if !ok { break }
			s.results = append(s.results, val)
		}
	}
	s.wg.Wait()
	//We're now done and you can retrieve the results.
}

// WORKER BEGINS HERE

func scraper_worker(wg *sync.WaitGroup, input chan ScraperTarget, output chan ScraperTarget, timeout int) {
	defer wg.Done()
	//Receive input from the channel until it is closed.
	targets := []ScraperTarget{}
	for {
		val,ok := <-input
		if !ok { break }
		targets = append(targets, val)
	}
	//Go through the targets and scrape each one.
	for index,_ := range targets {
		fqdn := targets[index].parent.fqdn
		scraped,err := scrape_url(targets[index].url.url, fqdn, timeout)
		if err == nil {
			targets[index].absolute_results = scraped[0]
			targets[index].relative_results = scraped[1]
		}
		targets[index].err = err
	}
	//Return results and close the channel.
	for _,result := range targets {
		output <-result
	}
	close(output)
}

// MISCELLANEOUS SCRAPING FUNCTIONS BEGIN HERE

func scrape_url(url string, fqdn string, timeout int) ([][]string,error) {
	//given a URL and an FQDN, scrape absolute and relative links on the page
	res,err := basic_request(url, timeout)
	if err != nil { return [][]string{},err }
	defer res.Body.Close()
	dom,err := ioutil.ReadAll(res.Body)
	if err != nil { return [][]string{},err }
	absolute_urls,err := get_fqdn_urls(string(dom), fqdn)
	if err != nil { return [][]string{},err }
	relative_urls,err := get_fqdn_urls(string(dom), "")
	if err != nil { return [][]string{},err }
	output := [][]string{absolute_urls, relative_urls}
	return output,nil
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

func get_html_urls(htmldata string) []string {
	//extract the URLs from HTML
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

// Code for extracting titles from HTML

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
