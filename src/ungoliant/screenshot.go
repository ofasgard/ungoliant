package main

import "os"
import "time"
import "os/exec"
import "context"
import "fmt"

func screenshot(url string, filepath string, chromepath string) error {
	//prepare args
	args := []string{"--headless", "--disable-gpu", "--hide-scrollbars", "--disable-crash-reporter", fmt.Sprintf("--screenshot=%s", filepath), url}
	if os.Geteuid() == 0 {
		args = append(args, "--no-sandbox")
	}
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


