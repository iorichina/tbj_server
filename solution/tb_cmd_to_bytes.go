package main

import (
	"fmt"
	"strconv"
	"strings"
)

func main() {
	ss := make([]string, 0)
	for _, s := range strings.Split("fe 86 e2 01 79 1d 09 34 3d", " ") {
		i, _ := strconv.ParseInt(s, 16, 16)
		ss = append(ss, strconv.FormatInt(i, 10))
	}
	fmt.Println(strings.Join(ss, ", "))
}
