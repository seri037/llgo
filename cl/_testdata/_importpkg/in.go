package main

import "github.com/goplus/llgo/cl/internal/stdio"

var hello = [...]int8{'H', 'e', 'l', 'l', 'o', '\n', 0}

func main() {
	stdio.Printf(&hello[0])
}
