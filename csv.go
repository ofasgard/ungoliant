package main

import "os"
import "io"
import "errors"
import "strings"
import "strconv"
import "encoding/csv"
import "fmt"

//Export a slice of Host objects into CSV format.

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

//Parse a CSV containing target hosts and ports.

func parse_csv(csvdata []byte) (map[string][]int,error) {
	output := make(map[string][]int)
	reader := csv.NewReader(strings.NewReader(string(csvdata)))
	for {
		record, err := reader.Read()
		if err == io.EOF { break }
		if err != nil { return output,err }
		if len(record) < 2 {
			return output,errors.New(fmt.Sprintf("Invalid record in CSV file: %s", record))
		}
		fqdn := record[0]
		port,err := strconv.Atoi(record[1])
		if err != nil {
			return output,errors.New(fmt.Sprintf("Invalid value for port: %s", record[1]))
		}
		if _,ok := output[fqdn]; ok {
			output[fqdn] = append(output[fqdn], port)
		} else {
			output[fqdn] = []int{port}
		}
	}
	return output,nil
}

func import_csv(csvdata []byte) ([]Host,error) {
	output := []Host{}
	parsed_hosts,err := parse_csv(csvdata)
	if err != nil { return output,err }
	for fqdn,ports := range parsed_hosts {
		for _,port := range ports {
			new_host := Host{}
			new_host.init(fqdn, port, false)
			output = append(output, new_host)
		}
	}
	return output,nil
}
