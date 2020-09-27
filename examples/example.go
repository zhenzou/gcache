package main

import (
	"fmt"

	"github.com/bluele/gcache"
)

func main() {
	gc := gcache.New(10).
		LFU().
		Build()
	gc.Set("key", "ok")

	v, err := gc.GetIFPresent("key")
	if err != nil {
		panic(err)
	}
	fmt.Println("value:", v)
}
