package main

import "time"
import "math/rand"

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
