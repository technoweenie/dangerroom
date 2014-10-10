package main

import (
	".."
	"fmt"
)

func main() {
	srv := dangerroom.NewServer(":8080")
	fmt.Println("Starting server on :8080")
	err := srv.ListenAndServe()
	fmt.Println(err)
}
