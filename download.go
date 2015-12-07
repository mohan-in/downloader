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
	NoOfSection    int
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
	pause chan int
	stop  chan int
}

func NewResource(url string) *Resource {
	res := &Resource{
		Url: url,
	}

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
	var NoOfSections int

	if res.Size>>20 < 50 {
		res.sectionSize = res.Size / int64(NoOfConnection)
		NoOfSections = NoOfConnection
	} else {
		res.sectionSize = int64(SectionSize) >> 20
		NoOfSections = int(res.Size / res.sectionSize)
	}

	res.sections = make([]Section, NoOfSections)
	for i := 0; i < NoOfSections; i++ {
		res.sections[i] = Section{
			Id:    i,
			data:  res.data[j : j+res.sectionSize],
			start: j,
			pause: make(chan int),
		}
		if i+1 == NoOfSections {
			res.sections[i].end = res.Size
		} else {
			j += res.sectionSize
			res.sections[i].end = j - 1
		}
	}

	return res
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
		select {
		case _ = <-s.pause:
			<-s.pause
		case _ = <-s.stop:
			return
		default:
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
	}

	logger.Printf("Section %d completed", s.Id)

	ticker.Stop()
	ch <- s.Id
}
