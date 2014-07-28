package main

import (
	"net/http"
	"net/url"
)

func main() {
	target, err := url.Parse("https://github.com/")
	if err != nil {
		panic(err)
	}

	proxy := NewSingleHostReverseProxy(target)
	proxy.LimitedBody = 5
	http.Handle("/", proxy)
	http.ListenAndServe(":8080", nil)
}
