package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/signal"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type users struct {
	Users []string `json:"users"`
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

func handler(w http.ResponseWriter, r *http.Request) {
	user := r.Header.Get("REMOTE-USER")
	uri := r.Header.Get("X-URI")
	host := r.Header.Get("X-Host")
	cert := r.Header.Get("X-Cert")

	log.Debug("Handling Request",
		zap.String("host", host),
		zap.String("uri", uri),
		zap.String("user", user))

	if user == "" || uri == "" || host == "" {
		w.WriteHeader(401)
		log.Error("REMOTE-USER, X-Forwarded-Proto, X-URI or X-Host Header not set")
		return
	}

	if cert != "" {
		certuser := getUserFromCert(cert)
		if certuser != "" {
			log.Info("Got User out of cert", zap.String("user", certuser))
		}
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
