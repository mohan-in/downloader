package main

import (
	"flag"
	"golang.org/x/net/websocket"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var (
	logger    *log.Logger
	resources []*Resource
	url       string
	daemon    bool
)

func init() {
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
		http.HandleFunc("/", indexHandler)
		http.HandleFunc("/static/", staticFilesHandler)
		http.HandleFunc("/resources", resourcesHandler)
		http.HandleFunc("/resources/pause", pauseHandler)
		http.HandleFunc("/resources/stop", stopHandler)
		http.Handle("/progress", websocket.Handler(progressHandler))
		http.ListenAndServe(":8080", nil)
	} else {
		res, err := NewResource(url, 0)
		if err != nil {
			logger.Println(err)
			return
		}

		done := make(chan int)

		for _, s := range res.Sections {
			s := s
			go s.Download(res.Url, done)
			go func() {
				for _ = range time.Tick(5 * time.Second) {
					logger.Printf("Section: %d; speed: %d KB/s; %% complete: %d", s.Id, s.Speed, s.PctComplete)
				}
			}()
		}

		for i := 0; i < len(res.Sections); i++ {
			logger.Printf("Section %d completed", <-done)
		}

		ioutil.WriteFile("file", res.data, os.ModePerm)
	}
}

func resourcesHandler(rw http.ResponseWriter, req *http.Request) {
	if req.Method == "POST" {
		url := req.FormValue("URL")

		res, err := NewResource(url, len(resources))
		if err != nil {
			logger.Println(err)
			return
		}
		resources = append(resources, res)

		done := make(chan int)
		go func() {
			for i := 0; i < len(res.Sections); i++ {
				<-done
			}
			ioutil.WriteFile("file", res.data, os.ModePerm)
		}()

		for _, s := range res.Sections {
			s := s
			go s.Download(res.Url, done)
		}
	}
}

func staticFilesHandler(rw http.ResponseWriter, req *http.Request) {
	http.ServeFile(rw, req, req.URL.Path[1:])
}

func indexHandler(rw http.ResponseWriter, req *http.Request) {
	http.ServeFile(rw, req, "static/downloader.html")
}

func progressHandler(ws *websocket.Conn) {
	websocket.JSON.Send(ws, resources)

	for _ = range time.Tick(5 * time.Second) {
		if len(resources) != 0 {
			websocket.JSON.Send(ws, resources)
		}
	}
}

func stopHandler(rw http.ResponseWriter, req *http.Request) {
	id, _ := strconv.Atoi(req.FormValue("id"))
	for i, r := range resources {
		if r.Id == id {
			r.Stop()
			resources = append(resources[:i], resources[i+1:]...)
			return
		}
	}
}

func pauseHandler(rw http.ResponseWriter, req *http.Request) {
	id, _ := strconv.Atoi(req.FormValue("id"))
	for _, r := range resources {
		if r.Id == id {
			r.Pause()
			return
		}
	}
}
