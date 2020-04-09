package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"unicode"
)

type users struct {
	Users    []string `json:"users"`
	Location string   `json:"location"`
}

type locations struct {
	Location map[string]users `json:"locations"`
}

// Config is the structure of the global Configuration object
type Config struct {
	Hosts  map[string]locations `json:"hosts"`
	Admins []string             `json:"admins"`
}

// Configuration is the globale config from /etc/auth-responder/config.json
var Configuration Config

func loadConfig(configFile string) (Config, error) {
	var config Config
	content, _ := ioutil.ReadFile(configFile)
	json.Unmarshal([]byte(content), &config)
	return config, nil
}

func setupSocket(socketPath string) (listener net.Listener) {
	os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatalf("Could not listen on %s: %v", socketPath, err)
		return
	}
	os.Chmod(socketPath, 0770)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		os.Remove(socketPath)
		os.Exit(0)
	}()
	return listener
}

func httpListener(listener net.Listener) {
	defer listener.Close()
	err := http.Serve(listener, nil)
	if err != nil {
		log.Fatalf("Could not start HTTP server: %v", err)
	}
}

func sanitizeUser(s string) string {
	return strings.Map(
		func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		},
		s,
	)
}

func isAdmin(user string) bool {
	for _, admin := range Configuration.Admins {
		if user == admin {
			return true
		}
	}
	//log.Printf("user: %s is not an Admin\n", user)
	return false
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func isAuthorized(user string, uri string, host string) bool {
	if isAdmin(user) {
		return true
	}

	if locations, ok := Configuration.Hosts[host]; ok {
		for location, users := range locations.Location {
			if strings.HasPrefix(location, uri) {
				fmt.Printf("%s is prefix of %s\n", uri, location)
				if stringInSlice(user, users.Users) {
					fmt.Println("users: ", users.Users)
					return true
				}
			}
		}
	}

	log.Printf("user: %s not Authorized\n", user)
	return false
}

func handler(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("REMOTE-USER")
	uri := r.Header.Get("X-URI")
	host := r.Header.Get("X-Host")

	fmt.Printf("host: %s uri: %s user: %s\n", host, uri, user)

	if user == "" || uri == "" || host == "" {
		w.WriteHeader(401)
		log.Printf("REMOTE-USER, X-Forwarded-Proto, X-URI or X-Host Header not set")
		return
	}

	if isAuthorized(user, uri, host) {
		w.WriteHeader(200)
		return
	}

	w.WriteHeader(403)
}

func main() {
	file := "/run/auth-responder/socket"
	confFile := "/etc/auth-responder/config.json"
	var err error

	Configuration, err = loadConfig(confFile)
	if err != nil {
		log.Fatalf("Could not read config %s: %v", confFile, err)
		return
	}

	listener := setupSocket(file)
	http.HandleFunc("/", handler)
	log.Printf("Start Listening on: %s", file)
	httpListener(listener)
}
