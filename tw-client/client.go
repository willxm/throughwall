package main

import (
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
	"strings"
	"syscall"
	"time"

	"github.com/willxm/throughwall/config"
	"github.com/willxm/throughwall/util"
	"golang.org/x/net/proxy"
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

	log.Println("Local Server Listening ", *port)
	log.Fatal(http.ListenAndServe(":"+*port, c))
}

type Client struct {
	RemoteAddr string
	Password   string
}

func (c *Client) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	socks, err := proxy.SOCKS5("tcp", c.RemoteAddr,
		nil,
		&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 10 * time.Second,
		},
	)

	if err != nil {
		log.Panic(err)
	}

	httpTransport := &http.Transport{
		Dial:            socks.Dial,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	httpClient := &http.Client{Transport: httpTransport}

	// we need to buffer the body if we want to read it here and send it
	// in the request.
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// you can reassign the body if you need to parse it as multipart
	r.Body = ioutil.NopCloser(bytes.NewReader(body))

	// create a new url from the raw RequestURI sent by the client
	// url := fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)
	uri := r.RequestURI
	if strings.Contains(uri, ":443") {
		uri = "https://" + strings.Split(uri, ":")[0]
	}

	log.Println(uri)

	proxyReq, err := http.NewRequest(r.Method, uri, bytes.NewReader(body))

	// We may want to filter some headers, otherwise we could just use a shallow copy
	// proxyReq.Header = req.Header
	proxyReq.Header = make(http.Header)
	for h, val := range r.Header {
		proxyReq.Header[h] = val
	}

	resp, err := httpClient.Do(proxyReq)

	if err != nil {
		log.Panic(err)
	}

	defer resp.Body.Close()
	for k, v := range resp.Header {
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
