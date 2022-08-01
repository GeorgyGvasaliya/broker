package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
	"time"
)

var (
	msgStorage    = make(map[string][]string)
	requestsQueue []*http.Request
	mu            sync.RWMutex
)

func getMessage(mp map[string][]string, key string, timeout time.Duration, r *http.Request) (string, error) {
	msgChan := make(chan string, 1)

	// костыль.. тк выбор кейсов рандомен
	duration := time.Second * timeout
	if timeout == 0 {
		duration = time.Microsecond
	}

	timeoutChan := time.After(duration)
	for {
		select {
		case value := <-msgChan:
			return value, nil
		case <-timeoutChan:
			mu.Lock()
			requestsQueue = requestsQueue[1:]
			mu.Unlock()
			return "", errors.New("No message found")
		default:
			mu.Lock()
			if item, ok := mp[key]; ok && len(item) > 0 && requestsQueue[0] == r {
				requestsQueue = requestsQueue[1:]
				mp[key] = mp[key][1:]
				msgChan <- item[0]
			}
			mu.Unlock()
		}
	}
}

func handleMessage(w http.ResponseWriter, r *http.Request) {
	key := r.URL.Path

	switch r.Method {
	case "GET":
		requestsQueue = append(requestsQueue, r)
		timeout := r.URL.Query().Get("timeout")
		timeoutInt, _ := strconv.Atoi(timeout)

		item, err := getMessage(msgStorage, key, time.Duration(timeoutInt), r)
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		_, err = fmt.Fprintf(w, item)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	case "PUT":
		v := r.URL.Query().Get("v")
		if v == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		msgStorage[key] = append(msgStorage[key], v)
		mu.Unlock()
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func main() {
	port := flag.String("p", "8080", "service port")
	flag.Parse()
	addr := ":" + *port

	http.HandleFunc("/", handleMessage)
	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}
