package main

import (
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/willxm/throughwall/config"
	"github.com/willxm/throughwall/util"
)

const (
	keyPath  = "./keys/server.key"
	certPath = "./keys/server.crt"
)

var port = flag.String("port", "6677", "Listen port.")

func main() {

	//graceful exit
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGUSR1, syscall.SIGUSR2)
	go util.SigHandler(ch)

	flag.Parse()

	cfg, err := config.Config()
	if err != nil {
		panic(err)
	}

	s := &Server{
		Password: cfg.Password,
	}

	log.Println("server listening ", *port)
	log.Fatal(http.ListenAndServeTLS(":"+*port, certPath, keyPath, s))
}

var transport = &http.Transport{
	ResponseHeaderTimeout: 30 * time.Second,
}

type Server struct {
	Password string
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	realurl := r.Header.Get("realto")

	su := strings.Split(realurl, ":")
	if len(su) > 1 && su[1] == "443" {
		realurl = "https://" + su[0]
	}

	log.Println(realurl)

	req, err := http.NewRequest(r.Method, realurl, r.Body)
	if err != nil {
		panic(err)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		panic(err)
	}
	for k, v := range resp.Header {

		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}

	data, _ := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	w.Write(data)
}
