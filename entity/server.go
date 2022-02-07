package entity

import "net/http"

type MyServer struct {
	http.Server
	shutdownReq chan bool
	reqCount    uint32
}
