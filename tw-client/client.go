package main

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/willxm/throughwall/config"
	"github.com/willxm/throughwall/cryptogram"
	"github.com/willxm/throughwall/util"
)

var port = flag.String("port", "1077", "Listen port.")
var remote = flag.String("remoteAddr", "127.0.0.1:6677", "Remote server address.")

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

	c := &Client{
		RemoteAddr: *remote,
		Password:   cfg.Password,
	}

	log.Println("server listening ", *port)
	log.Fatal(http.ListenAndServe(":"+*port, c))
}

type Client struct {
	RemoteAddr string
	Password   string
}

func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	conn, err := net.Dial("tcp", c.RemoteAddr)
	defer conn.Close()

	if err != nil {
		return
	}
	conns := tls.Client(conn, &tls.Config{InsecureSkipVerify: true})

	reader := bufio.NewReader(conns)

	log.Println(r.RequestURI)

	requestBody, err := ioutil.ReadAll(reader)

	if err != nil {
		panic(err)
	}

	reqData, err := cryptogram.AesEncrypt(requestBody, []byte(c.Password))
	if err != nil {
		panic(err)
	}

	bf := bytes.NewBuffer(reqData)

	r.Body = ioutil.NopCloser(bf)

	r.Write(conns)

	resp, err := http.ReadResponse(reader, r)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	for k, v := range resp.Header {
		for _, vv := range v {
			data, err := cryptogram.AesDecrypt([]byte(vv), []byte(c.Password))
			if err != nil {
				panic(err)
			}
			w.Header().Add(k, string(data))
		}
	}
	w.WriteHeader(resp.StatusCode)

	resqbody, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		panic(err)
	}

	respData, err := cryptogram.AesDecrypt(resqbody, []byte(c.Password))
	if err != nil {
		panic(err)
	}

	rbf := bytes.NewBuffer(respData)

	resp.Body = ioutil.NopCloser(rbf)

	io.Copy(w, resp.Body)
}
