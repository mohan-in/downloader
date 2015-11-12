package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	client http.Client
)

type Resource struct {
	url         string
	data        []byte
	size        int64
	sectionSize int64
	sections    []Section
	fileName    string
}

type Section struct {
	id    int
	start int64
	end   int64
	data  []byte
	speed int64
}

func (res *Resource) Download() {
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

	var j int64 = 0
	res.sections = make([]Section, 5)
	for i := 0; i < 5; i++ {
		res.sections[i] = Section{
			id:    i,
			data:  res.data[j : j+res.sectionSize],
			start: j,
		}
		j += res.sectionSize
		res.sections[i].end = j - 1
	}
}

func (s *Section) Download(url string, ch chan int) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		logger.Println(err)
	}

	req.Header.Add("Range", fmt.Sprintf("bytes=%d-%d", s.start, s.end))

	resp, err := client.Do(req)
	if err != nil {
		logger.Println(err)
	}

	defer resp.Body.Close()

	var bufSize, sectionSize int64

	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for _ = range ticker.C {
			s.speed = bufSize / (1024 * 5)
			bufSize = 0
		}
	}()

	buf := make([]byte, 128*1024)
	for {
		n, err := resp.Body.Read(buf)

		copy(s.data[sectionSize:], buf[0:n])
		sectionSize += int64(n)

		bufSize = bufSize + int64(n)

		if err == io.EOF {
			break
		}
	}

	logger.Printf("Section %d completed", s.id)

	ticker.Stop()
	ch <- s.id
}
