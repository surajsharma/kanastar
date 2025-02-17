package utils

import (
	"log"
	"time"
)

func Sleep(origin string, d time.Duration) {
	log.Printf("[💤][%s] sleeping for %d seconds", origin, d)
	time.Sleep(d * time.Second)
}
