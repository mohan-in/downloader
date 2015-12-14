package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var (
	logger *log.Logger
	url    string
	daemon bool
)

func init() {
	client = http.Client{}
	logger = log.New(os.Stdout, "downloader: ", log.Lshortfile)
	flag.BoolVar(&daemon, "daemon", false, "launch as daemon")
	flag.StringVar(&url, "file", "", "the file to download")
	flag.IntVar(&NoOfConnection, "n", 5, "Number of connections to the server")
	flag.IntVar(&SectionSize, "size", 50, "Section size in MB")
	flag.IntVar(&NetworkSpeed, "speed", 128, "Network speed in KB")
}

func main() {
	flag.Parse()

	if daemon {
		http.HandleFunc("/", downloadHandler)
		http.HandleFunc("/static/", staticFilesHandler)
		http.ListenAndServe(":8080", nil)
	} else {
		res, err := NewResource(url)
		if err != nil {
			logger.Println(err)
			return
		}

		ch := make(chan int)

		for _, s := range res.sections {
			s := s
			go s.Download(res.Url, ch)
			go listen(&s)
		}

		for i := 0; i < len(res.sections); i++ {
			<-ch
		}

		ioutil.WriteFile("file", res.data, os.ModePerm)
	}
}

func listen(s *Section) {
	ticker := time.NewTicker(5 * time.Second)
	for _ = range ticker.C {
		logger.Printf("Section: %d; speed: %d KB/s", s.Id, s.Speed)
	}
}

func staticFilesHandler(rw http.ResponseWriter, req *http.Request) {
	http.ServeFile(rw, req, req.URL.Path[1:])
}

func downloadHandler(rw http.ResponseWriter, req *http.Request) {
	http.ServeFile(rw, req, "static/downloader.html")
}
