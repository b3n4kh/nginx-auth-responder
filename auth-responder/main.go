package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"unicode"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
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
var configuration Config

// Logger is the global logging var
var log *zap.Logger
var logcfg zap.Config

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
		log.Fatal("Could not create socket",
			zap.String("socket", socketPath),
			zap.Error(err))
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
		log.Fatal("Could not start HTTP server", zap.Error(err))
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
	for _, admin := range configuration.Admins {
		if user == admin {
			return true
		}
	}
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

	if locations, ok := configuration.Hosts[host]; ok {
		for location, users := range locations.Location {
			if strings.HasPrefix(location, uri) {
				log.Debug("URI matches LocationRule",
					zap.String("uri", uri),
					zap.String("locationRule", location))
				if stringInSlice(user, users.Users) {
					log.Debug("Authenticating user",
						zap.String("user", user),
						zap.Strings("allowedUsers", users.Users))
					return true
				}
			}
		}
	}

	log.Info("user not Authorized", zap.String("user", user))
	return false
}

func handler(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("REMOTE-USER")
	uri := r.Header.Get("X-URI")
	host := r.Header.Get("X-Host")

	log.Debug("Handling Request",
		zap.String("host", host),
		zap.String("uri", uri),
		zap.String("user", user))

	if user == "" || uri == "" || host == "" {
		w.WriteHeader(401)
		log.Error("REMOTE-USER, X-Forwarded-Proto, X-URI or X-Host Header not set")
		return
	}

	if isAuthorized(user, uri, host) {
		w.WriteHeader(200)
		return
	}

	w.WriteHeader(403)
}

func main() {
	var (
		socketFile string
		confFile   string
		debug      bool
		err        error
	)

	flag.StringVar(&socketFile, "socket", "/run/auth-responder/socket", "The TCP Socket to open")
	flag.StringVar(&confFile, "config", "/etc/auth-responder/config.json", "The config File to read in")
	flag.BoolVar(&debug, "debug", false, "Run in Debug mode")
	flag.Parse()

	if !debug {
		logcfg = zap.NewProductionConfig()
	} else {
		logcfg = zap.NewDevelopmentConfig()
		logcfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	logger, _ := logcfg.Build()
	defer logger.Sync()

	log = logger

	configuration, err = loadConfig(confFile)
	if err != nil {
		log.Fatal("Could not read config",
			zap.String("file", confFile),
			zap.Error(err))
		return
	}

	listener := setupSocket(socketFile)
	http.HandleFunc("/", handler)
	log.Info("Started Listening", zap.String("socket", socketFile))
	httpListener(listener)
}
