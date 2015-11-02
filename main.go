package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
)

var client = http.Client{}
var maxConn = 2

func main() {
	var url string = "http://mirrors.mit.edu/pub/OpenBSD/5.8/i386/bsd.rd"

	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		fmt.Println(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	length := resp.ContentLength
	fmt.Println(length)

	var j, sectionSize int64
	sectionSize = length / 5

	data := make([]byte, length)
	ch := make(chan int)

	j = 0
	for i := 0; i < maxConn; i++ {
		go download(url, j, j+sectionSize, data[j:j+sectionSize], ch)
		j += sectionSize
	}

	ioutil.WriteFile("file", data, os.ModePerm)
}

func download(url string, start, end int64, data []byte, ch chan int) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Range", "bytes="+strconv.FormatInt(start, 10)+"-"+strconv.FormatInt(end, 10))
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)

	for i, c := range buf {
		data[i] = c
	}

	ch <- 0
}
