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

var msgStorage = make(map[string][]string)
var mu sync.RWMutex

func getMessage(mp map[string][]string, key string, timeout time.Duration) (string, error) {
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
			return "", errors.New("No item found")
		default:
			mu.Lock()
			if item, ok := mp[key]; ok && len(item) > 0 {
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
		timeout := r.URL.Query().Get("timeout")
		timeoutInt, _ := strconv.Atoi(timeout)

		item, err := getMessage(msgStorage, key, time.Duration(timeoutInt))
		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		fmt.Fprintf(w, item)
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
