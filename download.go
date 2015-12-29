package main

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

var (
	NoOfConnection int = 5
	SectionSize    int = 50
	NetworkSpeed   int = 128
)

type Resource struct {
	Id          int
	Url         string
	data        []byte
	Size        int64
	sectionSize int64
	Sections    []*Section
	FileName    string
}

type Section struct {
	Id          int
	start       int64
	end         int64
	data        []byte
	Speed       int64
	PctComplete int
	pause       chan int
	stop        chan int
}

var client http.Client = http.Client{}

func NewResource(url string) (*Resource, error) {
	res := &Resource{
		Url: url,
	}

	//find out the size of resource
	resp, err := http.Head(res.Url)
	if err != nil {
		logger.Println(err)
		return nil, err
	}
	res.Size = resp.ContentLength
	res.data = make([]byte, res.Size)

	//determine number of sections
	var NoOfSections int
	if res.Size>>20 < 50 {
		res.sectionSize = res.Size / int64(NoOfConnection)
		NoOfSections = NoOfConnection
	} else {
		res.sectionSize = int64(SectionSize) >> 20
		NoOfSections = int(res.Size / res.sectionSize)
	}

	//create sections
	var j int64
	res.Sections = make([]*Section, NoOfSections)
	for i := 0; i < NoOfSections; i++ {
		res.Sections[i] = &Section{
			Id:    i,
			data:  res.data[j : j+res.sectionSize],
			start: j,
			pause: make(chan int),
		}
		if i+1 == NoOfSections {
			res.Sections[i].end = res.Size
		} else {
			j += res.sectionSize
			res.Sections[i].end = j - 1
		}
	}

	return res, nil
}

func (r *Resource) Stop() {
	for _, s := range r.Sections {
		s.stop <- 0
	}
}

func (r *Resource) Pause() {
	for _, s := range r.Sections {
		s.pause <- 0
	}
}

func (s *Section) Download(url string, done chan int) {
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

	//calculate the speed every five seconds
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	go func() {
		for _ = range ticker.C {
			s.Speed = bufSize / 5
			bufSize = 0
			s.PctComplete = int(sectionSize * 100 / (s.end - s.start))
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

			if err != nil {
				if err == io.EOF {
					done <- s.Id
					s.PctComplete = 100
					s.Speed = 0
					return
				} else {
					logger.Printf("Error in downloading section %d. Restartinf download", s.Id)
					s.start += bufSize
					go s.Download(url, done)
					return
				}
			}

			copy(s.data[sectionSize:], buf[0:n])
			sectionSize += int64(n)
			bufSize = bufSize + int64(n)
		}
	}
}
