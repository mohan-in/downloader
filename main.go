package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	client http.Client
	logger *log.Logger
	url    string
)

type resource struct {
	url         string
	data        []byte
	size        int64
	sectionSize int64
	sections    []section
	fileName    string
}

type section struct {
	id    int
	start int64
	end   int64
	data  []byte
}

func init() {
	client = http.Client{}
	logger = log.New(os.Stdout, "downloader: ", log.Lshortfile)
	flag.StringVar(&url, "file", "", "the file to download")
}

func main() {
	flag.Parse()

	res := &resource{
		url: url,
	}

	res.download()
}

func (res *resource) download() {
	req, err := http.NewRequest("HEAD", res.url, nil)
	if err != nil {
		logger.Println(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Println(err)
	}

	res.size = resp.ContentLength
	res.sectionSize = res.size / 5
	res.data = make([]byte, res.size)

	ch := make(chan int)

	var j int64 = 0
	res.sections = make([]section, 5)
	for i := 0; i < 5; i++ {
		res.sections[i] = section{
			id:    i,
			data:  res.data[j : j+res.sectionSize],
			start: j,
		}
		j += res.sectionSize
		res.sections[i].end = j - 1
	}

	for _, s := range res.sections {
		s := s
		go s.download(res.url, ch)
	}

	for i := 0; i < 5; i++ {
		<-ch
	}

	ioutil.WriteFile("file", res.data, os.ModePerm)
}

func (s *section) download(url string, ch chan int) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Println(err)
	}

	req.Header.Add("Range", "bytes="+strconv.FormatInt(s.start, 10)+"-"+strconv.FormatInt(s.end, 10))

	resp, err := client.Do(req)
	if err != nil {
		logger.Println(err)
	}

	defer resp.Body.Close()

	var n, size int64

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for _ = range ticker.C {
			logger.Println("Section: " + strconv.Itoa(s.id) + "; speed: " + strconv.FormatInt(n/(1024*5), 10) + "KB/s")
			n = 0
		}
	}()

	buf := make([]byte, 128*1024)
	for {
		tn, err := resp.Body.Read(buf)

		copy(s.data[size:], buf[0:tn])
		size += int64(tn)

		n = n + int64(tn)

		if err == io.EOF {
			ticker.Stop()
			break
		}
	}

	logger.Println("Section " + strconv.Itoa(s.id) + " completed")

	ch <- 0
}
