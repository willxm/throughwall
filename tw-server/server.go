package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/willxm/throughwall/cryptogram"

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

	//decode
	data, err := ioutil.ReadAll(r.Body)
	if err != nil {
		panic(err)
	}
	decryptReq, err := cryptogram.AesDecrypt(data, []byte(s.Password))
	if err != nil {
		panic(err)
	}

	bf := bytes.NewBuffer(decryptReq)

	r.Body = ioutil.NopCloser(bf)

	req, err := http.NewRequest(r.Method, r.URL.String(), r.Body)
	resp, err := transport.RoundTrip(req)
	for k, v := range resp.Header {
		for _, vv := range v {
			data, err := cryptogram.AesEncrypt([]byte(vv), []byte(s.Password))
			if err != nil {
				panic(err)
			}
			w.Header().Add(k, string(data))
		}
	}
	if err != nil {
		panic(err)
	}

	log.Println("test conn")

	resqData, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	encryptResp, err := cryptogram.AesEncrypt(resqData, []byte(s.Password))
	if err != nil {
		panic(err)
	}

	rbf := bytes.NewBuffer(encryptResp)

	resp.Body = ioutil.NopCloser(rbf)

	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	w.Write(data)
}
