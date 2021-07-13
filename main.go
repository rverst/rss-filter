package main

import (
	"github.com/integrii/flaggy"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	envAddress = "LISTEN_ADDR"
	envUser = "AUTH_USER"
	envPassword = "AUTH_PASSWORD"
	envDisableAuth = "DISABLE_AUTH"
	defaultAddress = ":80"
)

var (
	version = "dev"
	handler *rssHandler
)

func main() {

	log.Logger = zerolog.New(zerolog.NewConsoleWriter()).With().Timestamp().Str("version", version).Caller().Logger()

	address := defaultAddress
	authUser := ""
	authPass := ""
	disableAuth := false
	flaggy.SetVersion(version)
	flaggy.String(&address, "a", "address", "The local address the server listens on, in the for <address>:<port>.")
	flaggy.String(&authUser, "u", "auth_user", "User part for basic http authentication of the endpoint.")
	flaggy.String(&authPass, "p", "auth_password", "Secret part for basic http authentication of the endpoint.")
	flaggy.Bool(&disableAuth, "", "disable_auth", "Disable authentication.")
	flaggy.Parse()

	adr := os.Getenv(envAddress)
	user := os.Getenv(envUser)
	pass := os.Getenv(envPassword)
	disA := os.Getenv(envDisableAuth)
	if adr != "" && (address == defaultAddress || address == "") {
		address = adr
	}
	if user != "" && authUser == "" {
		authUser = user
	}
	if pass != "" && authPass == "" {
		authPass = pass
	}
	if disA != "" && !disableAuth {
		var err error
		disableAuth, err = strconv.ParseBool(disA)
		if err != nil {
			log.Fatal().Err(err).Msg("can't parse " + envDisableAuth)
		}
	}

	if authPass == "" && !disableAuth {
		log.Fatal().Msg("you MUST provide a password")
	}

	handler = newRssHandler(authUser, authPass, disableAuth)
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

