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

	http.Handle("/", NewSingleHostReverseProxy(target))
	http.ListenAndServe(":8080", nil)
}
