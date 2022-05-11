package main

import "os"
import "math/rand"


//Create a directory of it doesn't exist.

func create_dir(name string) error {
	_, err := os.Stat(name)
	if os.IsNotExist(err) {
		err = os.MkdirAll(name, 0755)
		return err
	}
	return nil
}

//Generate a random string of determinate length.

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func random_string(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

//Check if a slice contains an value.

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

func string_in_slice(list []string, search string) bool {
	valid := make(map[string]bool)
	for _,val := range list {
		valid[val] = true
	}
	if valid[search] {
		return true
	}
	return false
}

//Check if a slice contains ONLY one value.

func int_slice_equal(list ...int) bool {
	if len(list) == 0 { return true }
	search := list[0]
	valid := true
	for _,val := range list {
		if val != search { valid = false}
	}
	return valid
}

func string_slice_equal(list ...string) bool {
	if len(list) == 0 { return true }
	search := list[0]
	valid := true
	for _,val := range list {
		if val != search { valid = false}
	}
	return valid
}
