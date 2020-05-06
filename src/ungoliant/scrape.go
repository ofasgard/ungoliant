package main

import "strings"
import "sync"
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

func scrape_worker(fqdn string, baseurl string, timeout int, jobs chan Url, results chan []string, wg *sync.WaitGroup) {
	//worker function to scrape multiple URLs and return the results
	defer wg.Done()
	defer close(results)
	for job := range jobs {
		res,err := scrape_url(job.url, fqdn, timeout)
		if err == nil {
			output := []string{}
			for _,link := range res[0] { output = append(output, link) }
			for _,uri := range res[1] {
				link := baseurl + uri
				output = append(output, link)
			}
			results <- output
		}
	}
	
}

func scrape_host(target *Host, timeout int, threads int) {
	var scraper_wg sync.WaitGroup
	//create job lists
	job_lists := [][]Url{}
	for len(job_lists) < threads {
		job_lists = append(job_lists, []Url{})
	}
	//populate job lists with URLs
	i := 0
	for index,_ := range target.urls {
		job_lists[i] = append(job_lists[i], target.urls[index])
		i++
		if i == threads {
			i = 0
		}
	}
	//create jobs
	result_list := []chan []string{}
	result_count := []int{}
	for index,list := range job_lists {
		scraper_wg.Add(1)
		jobs := make(chan Url, len(list))
		results := make(chan []string, len(list))
		go scrape_worker(target.fqdn, target.base_url(), timeout, jobs, results, &scraper_wg)
		for _,job := range job_lists[index] {
			jobs <- job
		}
		close(jobs)
		result_list = append(result_list,results)
		result_count = append(result_count,len(list))
	}
	//wait for all workers to return
	scraper_wg.Wait()
	for index,results := range result_list {
		for a := 0; a < result_count[index]; a++ {
			for _,link := range <-results {
				target.add_url(link)
			}
		}
	}
}

