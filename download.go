package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	client         http.Client
	NoOfConnection int = 5
	SectionSize    int = 50
	NetworkSpeed   int = 128
)

type Resource struct {
	Url         string
	data        []byte
	Size        int64
	sectionSize int64
	sections    []Section
	FileName    string
}

type Section struct {
	Id    int
	start int64
	end   int64
	data  []byte
	Speed int64
}

func (res *Resource) Download() {
	req, err := http.NewRequest("HEAD", res.Url, nil)
	if err != nil {
		logger.Println(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Println(err)
	}

	res.Size = resp.ContentLength
	res.data = make([]byte, res.Size)

	var j int64
	var noOfSections int

	if res.Size>>20 < 50 {
		res.sectionSize = res.Size / int64(NoOfConnection)
		noOfSections = NoOfConnection
	} else {
		res.sectionSize = int64(SectionSize) >> 20
		noOfSections = int(res.Size / res.sectionSize)
	}

	res.sections = make([]Section, noOfSections)
	for i := 0; i < noOfSections; i++ {
		res.sections[i] = Section{
			Id:    i,
			data:  res.data[j : j+res.sectionSize],
			start: j,
		}
		if i+1 == noOfSections {
			res.sections[i].end = res.Size
		} else {
			j += res.sectionSize
			res.sections[i].end = j - 1
		}
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
			s.Speed = bufSize / (1024 * 5)
			bufSize = 0
		}
	}()

	buf := make([]byte, NetworkSpeed<<10)
	for {
		n, err := resp.Body.Read(buf)

		copy(s.data[sectionSize:], buf[0:n])
		sectionSize += int64(n)
		bufSize = bufSize + int64(n)

		if err != nil {
			if err == io.EOF {
				break
			} else {
				logger.Printf("Error in downloading section %d. Restartinf download", s.Id)
				s.start += bufSize
				go s.Download(url, ch)
				return
			}
		}
	}

	logger.Printf("Section %d completed", s.Id)

	ticker.Stop()
	ch <- s.Id
}
