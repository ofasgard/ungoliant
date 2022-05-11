package main

import "sync"
import "math/rand"
import "fmt"
import "time"

func bruteforce(proxy bool, timeout int, threads int, hosts []Host) []Host {
	//initial bruteforce without proxy
	bf := Bruteforcer{}
	bf.init(false, timeout, threads)
	for index,_ := range hosts {
		for urlindex,_ := range hosts[index].urls {
			if !hosts[index].urls[urlindex].retrieved {
				bf.add(&hosts[index].urls[urlindex])
			}
		}
	}
	go bf.run()
	for !bf.finished {
		time.Sleep(5 * time.Second)
		fmt.Printf("\t...%d completed, %d errors\n", bf.progress, bf.errors)
	}
	//flush URLs
	for index,_ := range hosts {
		hosts[index].flush_urls()
	}
	//run through proxy
	if proxy {
		bf.init(true, timeout, threads)
		for index,_ := range hosts {
			for urlindex,_ := range hosts[index].urls {
				if !hosts[index].urls[urlindex].retrieved_proxy {
					bf.add(&hosts[index].urls[urlindex])
				}
			}
		}
		go bf.run()
		for !bf.finished {
			time.Sleep(5 * time.Second)
			fmt.Printf("\t...%d completed, %d errors (via proxy)\n", bf.progress, bf.errors)
		}
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

	finished bool
	progress int
	errors int
}

func (b *Bruteforcer) init(proxy bool, timeout int, threads int) {
	b.proxy = proxy
	b.timeout = timeout
	b.threads = threads
	
	b.targets = []*Url{}

	b.finished = false
	b.progress = 0
	b.errors = 0
}

func (b *Bruteforcer) add(target *Url) {
	b.targets = append(b.targets, target)
}

func (b *Bruteforcer) run() {
	//Randomise the order of the target list.
	rand.Shuffle(len(b.targets), func(i, j int) { b.targets[i], b.targets[j] = b.targets[j], b.targets[i] })
	//Create a number of worker goroutines equal to the thread limit.
	input_channels := []chan *Url{}
	progress_channels := []chan bool{}
	for i := 0; i < b.threads; i++ {
		b.wg.Add(1)
		input_chan := make(chan *Url)
		input_channels = append(input_channels, input_chan)
		progress_chan := make(chan bool)
		progress_channels = append(progress_channels, progress_chan)
		go bruteforcer_worker(&b.wg, input_chan, progress_chan, b.proxy, b.timeout)
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
	//Update progress as workers return.
	for {
		closed := 0
		for _,progress_chan := range progress_channels {
			select {
				case x,ok := <-progress_chan:
					if ok {
						if x { b.progress++ }
						if !x { b.errors++ }
					} else {
						closed++
					}
				default:
					//pass
			}
		}
		if closed == len(progress_channels) { break }
	}
	//Wait for all waitgroups to complete.
	b.wg.Wait()
	b.finished = true
}

// WORKER BEGINS HERE

func bruteforcer_worker(wg *sync.WaitGroup, input chan *Url, progress chan bool, proxy bool, timeout int) {
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
		targets[index].retrieve(proxy, timeout)
		if targets[index].err == nil {
			progress <- true
		} else {
			progress <- false
		}
	}
	close(progress)
}
