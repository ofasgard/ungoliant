package main

import "sync"
import "math/rand"

func bruteforce(proxy bool, proxy_host string, proxy_port int, timeout int, threads int, hosts []Host) []Host {
	//initial bruteforce without proxy
	bf := Bruteforcer{}
	bf.init(false, proxy_host, proxy_port, timeout, threads)
	for index,_ := range hosts {
		for urlindex,_ := range hosts[index].urls {
			if !hosts[index].urls[urlindex].retrieved {
				bf.add(&hosts[index].urls[urlindex])
			}
		}
	}
	bf.run()
	//flush URLs
	for index,_ := range hosts {
		hosts[index].flush_urls()
	}
	//run through proxy
	if proxy {
		bf.init(true, proxy_host, proxy_port, timeout, threads)
		for index,_ := range hosts {
			for urlindex,_ := range hosts[index].urls {
				if !hosts[index].urls[urlindex].retrieved_proxy {
					bf.add(&hosts[index].urls[urlindex])
				}
			}
		}
		bf.run()
	}
	//return
	return hosts
}

// BRUTEFORCER BEGINS HERE

type Bruteforcer struct {
	proxy bool
	proxy_host string
	proxy_port int
	timeout int
	threads int

	targets []*Url
	wg sync.WaitGroup
}

func (b *Bruteforcer) init(proxy bool, proxy_host string, proxy_port int, timeout int, threads int) {
	b.proxy = proxy
	b.proxy_host = proxy_host
	b.proxy_port = proxy_port
	b.timeout = timeout
	b.threads = threads
	
	b.targets = []*Url{}
}

func (b *Bruteforcer) add(target *Url) {
	b.targets = append(b.targets, target)
}

func (b *Bruteforcer) run() {
	//Randomise the order of the target list.
	rand.Shuffle(len(b.targets), func(i, j int) { b.targets[i], b.targets[j] = b.targets[j], b.targets[i] })
	//Create a number of worker goroutines equal to the thread limit.
	input_channels := []chan *Url{}
	for i := 0; i < b.threads; i++ {
		b.wg.Add(1)
		input_chan := make(chan *Url)
		input_channels = append(input_channels, input_chan)
		go bruteforcer_worker(&b.wg, input_chan, b.proxy, b.proxy_host, b.proxy_port, b.timeout)
	}
	//Hand out URLs to all the goroutines we just created, using the input channels.
	current_channel := 0
	for index,_ := range b.targets {
		input_channels[current_channel] <- b.targets[index]
		current_channel++
		if current_channel == b.threads { current_channel = 0 }
	}
	//Close the input channels.
	for index,_ := range input_channels {
		close(input_channels[index])
	}
	//Wait for all waitgroups to complete.
	b.wg.Wait()
}

// WORKER BEGINS HERE

func bruteforcer_worker(wg *sync.WaitGroup, input chan *Url, proxy bool, proxy_host string, proxy_port int, timeout int) {
	defer wg.Done()
	//Receive input from the channel until it is closed.
	targets := []*Url{}
	for {
		val,ok := <-input
		if !ok { break }
		targets = append(targets, val)
	}
	//Go through the targets and retrieve each one.
	for index,_ := range targets {
		targets[index].retrieve(proxy, proxy_host, proxy_port, timeout)
	}
}
