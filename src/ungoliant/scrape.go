package main

func scrape_url(url string, fqdn string, chromepath string) ([]string,[]string,error) {
	//given a URL and an FQDN, scrape absolute and relative links on the page
	dom,err := chrome_request(url, chromepath)
	if err != nil { return []string{},[]string{},err }
	absolute_urls,err := chrome_get_urls(dom, fqdn)
	if err != nil { return []string{},[]string{},err }
	relative_urls,err := chrome_get_urls(dom, "")
	if err != nil { return []string{},[]string{},err }
	return absolute_urls,relative_urls,nil
}
