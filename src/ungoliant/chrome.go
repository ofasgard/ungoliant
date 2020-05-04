package main

import "os"
import "time"
import "os/exec"
import "context"
import "fmt"
import "bytes"
import "errors"
import "strings"
import "io/ioutil"

/* UTILITY FUNCTIONS */

func check_chrome(chromepath string) string {
	//check if chrome exists, using either specified path or default locations
	defaults := []string{
		"/usr/bin/chromium",
		"/usr/bin/chromium-browser",
		"/usr/bin/google-chrome-stable",
		"/usr/bin/google-chrome",
		"/Applications/Google Chrome.app/Contents/MacOS/Google Chrome",
		"/Applications/Google Chrome Canary.app/Contents/MacOS/Google Chrome Canary",
		"/Applications/Chromium.app/Contents/MacOS/Chromium",
		"C:/Program Files (x86)/Google/Chrome/Application/chrome.exe",
	}
	if chromepath != "" {
		_,err := os.Stat(chromepath)
		if !os.IsNotExist(err) {
			return chromepath
		}
	}
	for _,path := range defaults {
		_,err := os.Stat(path)
		if !os.IsNotExist(err) {
			return path
		}
	}
	return ""
}

func check_google_blocked(url string) bool {
	//checks if we are being blocked by Google
	res,err := basic_request(url, 5)
	if err != nil {
		return false //we're not blocked, we just can't connect
	}
	defer res.Body.Close()
	if res.StatusCode == 503 {
		return true //blocked!
	}
	detect_string := "Our systems have detected unusual traffic from your computer network."
	body, err := ioutil.ReadAll(res.Body)
	if err == nil {
		if strings.Contains(string(body), detect_string) {
			return true //blocked!
		}
	}
	return false
}

/* SCREENSHOT FUNCTIONALITY */

func screenshot(url string, filepath string, chromepath string) error {
	//takes a screenshot of a specified URL using headless Chrome
	args := []string{"--headless", "--disable-gpu", "--hide-scrollbars", "--disable-crash-reporter", "--ignore-certificate-errors", "--window-size=1600,900", "--virtual-time-budget=3000", fmt.Sprintf("--screenshot=%s", filepath)}
	if os.Geteuid() == 0 {
		args = append(args, "--no-sandbox")
	}
	args = append(args, url)
	//prepare context
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancel()
	//execute command
	cmd := exec.CommandContext(ctx, chromepath, args...)
	err := cmd.Start()
	if err != nil {
		return err
	}
	//wait to finish or error
	err = cmd.Wait()
	return err
}

/* GOOGLE DORK FUNCTIONALITY */

func chrome_request(url string, chromepath string) (string,error) {
	//makes a request and dumps the DOM using headless Chrome
	args := []string{"--headless", "--disable-gpu", "--disable-crash-reporter", "--ignore-certificate-errors", "--window-size=1600,900", "--virtual-time-budget=3000", "--dump-dom"}
	if os.Geteuid() == 0 {
		args = append(args, "--no-sandbox")
	}
	args = append(args, url)
	//prepare context
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(5)*time.Second)
	defer cancel()
	//execute command
	cmd := exec.CommandContext(ctx, chromepath, args...)
	var output bytes.Buffer
	cmd.Stdout = &output
	err := cmd.Start()
	if err != nil {
		return "",err
	}
	//wait to finish or error
	err = cmd.Wait()
	return output.String(),err
}

func chrome_dork(chromepath string, fqdn string, page_max int) ([]string, error) {
	//automatically perform Google dorking to get links associated with an FQDN
	output := []string{}
	page := 0
	for {
		target := fmt.Sprintf("https://www.google.co.uk/search?q=site:%s&start=%d", fqdn, page * 10)
		dom,err := chrome_request(target, chromepath)
		if err != nil { break }
		res,err := get_fqdn_urls(dom, fqdn)
		if err != nil { break }
		if len(res) == 0 { break }
		output = append(output, res...)
		if page == page_max { break }
		page++
	}
	//did we break because we're done, or because of a captcha?
	var err error
	check_url := fmt.Sprintf("https://www.google.co.uk/search?q=site:%s&start=%d", fqdn, page * 10)
	if check_google_blocked(check_url) {
		err = errors.New("Google is probably blocking you.")
	}
	return output,err
}




