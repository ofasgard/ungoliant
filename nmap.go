package main
//Contains functions for parsing NMap XML output into a map of hostnames/IP addresses and open ports.

import "encoding/xml"

/*
* parse_nmap (xmldata []byte) ([]Host,error)
*
* Parse an Nmap XML file to extract the information we care about.
* Takes a []byte containing raw XML as input.
* Returns a map of fqdns and open ports as output.
* i.e. "scanme.nmap.org": [22, 80, 9929, 31337]
*/

func parse_nmap(xmldata []byte) (map[string][]int,error) {
	output := make(map[string][]int)
	var parsed_scan NmapScan
	err := xml.Unmarshal(xmldata, &parsed_scan)
	if err != nil {
		return output,err
	}
	for _,host := range parsed_scan.Hosts {
		//figure out which ports are open
		ports := []int{}
		for _,port := range host.Ports.Ports {
			if (port.PortState.Status == "open") && (port.Proto == "tcp") {
				ports = append(ports, port.PortNo)
			}
			//add a host entry for each ip
			for _,address := range host.Addresses {
				output[address.IP] = ports
			}
			//add a host entry for each FQDN
			for _,hostname := range host.Hostnames.Hostnames {
				output[hostname.Name] = ports
			}
		}
	}
	return output,nil
}

func import_nmap(xmldata []byte) ([]Host,error) {
	output := []Host{}
	parsed_hosts,err := parse_nmap(xmldata)
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

// Structs used by encoding/xml to parse the Nmap XML.

type NmapScan struct {
	XMLName xml.Name `xml:"nmaprun"`
	Hosts []NmapHost `xml:"host"`
}

type NmapHost struct {
	XMLName xml.Name `xml:"host"`
	Addresses []NmapAddress `xml:"address"`
	Hostnames NmapHostnameList `xml:"hostnames"`
	Ports NmapPortList `xml:"ports"`
}

type NmapAddress struct {
	XMLName xml.Name `xml:"address"`
	IP string `xml:"addr,attr"`
}

type NmapHostnameList struct {
	XMLName xml.Name `xml:"hostnames"`
	Hostnames []NmapHostname `xml:"hostname"`
}

type NmapHostname struct {
	XMLName xml.Name `xml:"hostname"`
	Name string `xml:"name,attr"`
}

type NmapPortList struct {
	XMLName xml.Name `xml:"ports"`
	Ports []NmapPort `xml:"port"`
}

type NmapPort struct {
	XMLName xml.Name `xml:"port"`
	PortNo int `xml:"portid,attr"`
	Proto string `xml:"protocol,attr"`
	PortState NmapState `xml:"state"`
}

type NmapState struct {
	XMLName xml.Name `xml:"state"`
	Status string `xml:"state,attr"`
}


