package main

import (
	"blog/entity"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/gorilla/mux"
)

var users = []entity.User{
	{ID: "1", Name: "John", Age: 20},
	{ID: "2", Name: "Mary", Age: 30},
	{ID: "3", Name: "Mike", Age: 40},
}

type myServer struct {
	http.Server
	shutdownReq chan bool
	reqCount    uint32
}

func NewServer() *myServer {
	s := &myServer{
		Server: http.Server{
			Addr:         ":8000",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
		},
		shutdownReq: make(chan bool),
	}

	router := mux.NewRouter()
	router.HandleFunc("/", s.RootHandler)
	router.HandleFunc("/shutdown", s.ShutdownHandler)

	router.HandleFunc("/get-users", getUsers).Methods("GET")
	// router.HandleFunc("/add-post", addPosts).Methods("POST")

	log.Println("Server started on port:", s.Addr)

	s.Handler = router

	return s
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func (s *myServer) RootHandler(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Server is up and running...."))
}

func (s *myServer) ShutdownHandler(w http.ResponseWriter, r *http.Request) {
	if !atomic.CompareAndSwapUint32(&s.reqCount, 0, 1) {
		log.Printf("Shutdown call in progress...")
		return
	}

	go func() {
		s.shutdownReq <- true
	}()
}

func (s *myServer) WaitShutdown() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sig:
		atomic.CompareAndSwapUint32(&s.reqCount, 0, 1)
		log.Printf("Shutdown request (signal: %v)", sig)
	case sig := <-s.shutdownReq:
		log.Printf("Shutdown request (signal: %v)", sig)
	}
	log.Printf("Stopping http server ...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := s.Shutdown(ctx)
	if err != nil {
		log.Printf("Shutdown request error: %v", err)
	}
}

func main() {

	server := NewServer()
	done := make(chan bool)
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Printf("Listen and Serve: %v", err)
		}
		done <- true
	}()

	server.WaitShutdown()

	log.Println("DONE !!", <-done)
}
