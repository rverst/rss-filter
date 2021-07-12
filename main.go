package main

import (
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"time"
)

const (
	envAddress = "LISTEN_ADDR"
	envApiKey = "API_KEY"
	defaultAddress = ":80"
)

var (
	version = "dev"
	handler *rssHandler
)

func main() {

	log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Str("version", version).Caller().Logger()

	address := defaultAddress
	apiKey := ""
	disableApiKey := false
	flaggy.SetVersion(version)
	flaggy.String(&address, "a", "address", "The local address the server listens on, in the for <address>:<port>.")
	flaggy.String(&apiKey, "k", "api_key", "Secret key to protect the endpoint.")
	flaggy.Bool(&disableApiKey, "", "disable_api_key", "Disable the requirement of an api key.")
	flaggy.Parse()

	adr := os.Getenv(envAddress)
	key := os.Getenv(envApiKey)
	if adr != "" && (address == defaultAddress || address == "") {
		address = adr
	}
	if key != "" && apiKey == "" {
		apiKey = key
	}

	if apiKey == "" && !disableApiKey {
		log.Fatal().Msg("you MUST provide an api key")
	}

	handler = newRssHandler(apiKey)
	server := &http.Server{
		Addr:           address,
		Handler:        handler,
		ReadTimeout:    10 * time.Second,
		WriteTimeout:   10 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Info().Msgf("listening on: '%s'", address)
	if err := server.ListenAndServe(); err != nil {
		log.Fatal().Err(err).Send()
	}
}

