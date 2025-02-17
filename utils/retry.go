package utils

import (
	"fmt"
	"net/http"
)

func HTTPWithRetry(f func(string) (*http.Response, error), url string) (*http.Response, error) {
	count := 10
	var resp *http.Response
	var err error
	for i := 0; i < count; i++ {
		resp, err = f(url)
		if err != nil {
			fmt.Printf("[retry] error calling url %v\n", url)
			Sleep("retry", 5)
		} else {
			break
		}
	}
	return resp, err
}
