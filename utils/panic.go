package utils

import "log"

func HandlePanic(err string) {
	if r := recover(); r != nil {
		log.Printf("[panic] %s", err)
		log.Printf("[panic] recovered from panic: %v", r)
	}
}
