package main

import "os"
import "time"
import "strconv"
import "math/rand"
import "encoding/csv"

//Generates a random string of determinate length.


var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func random_string(length int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

//Check if a slice contains an integer.

func int_in_slice(list []int, search int) bool {
	valid := make(map[int]bool)
	for _,val := range list {
		valid[val] = true
	}
	if valid[search] {
		return true
	}
	return false
}

//Export a slice of WebResult or Host objects into CSV format.

func export_csv(filename string, records [][]string) error {
	fd,err := os.Create(filename)
	if err != nil {
		return err
	}
	w := csv.NewWriter(fd)
	for _,record := range records {
		err = w.Write(record)
		if err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

func webresults_to_csv(filename string, results []WebResult) error {
	records := [][]string{}
	records = append(records, []string{"Host", "Port", "Protocol", "Status Code", "Status Text"})
	for _,result := range results {
		var protocol string
		if result.https { protocol = "HTTPS" } else { protocol = "HTTP" }
		records = append(records, []string{result.fqdn, strconv.Itoa(result.port), protocol, strconv.Itoa(result.statuscode), result.statustext})
	}
	return export_csv(filename, records)
}

func hosts_to_csv(filename string, hosts []Host) error {
	records := [][]string{}
	records = append(records, []string{"Host", "Port", "Url", "Status Code", "Status Text"})
	for _,host := range hosts {
		for _,url := range host.urls {
			records = append(records, []string{host.fqdn, strconv.Itoa(host.port), url.url, strconv.Itoa(url.statuscode), url.statustext})
		}
	}
	return export_csv(filename, records)
}
