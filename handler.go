package main

import (
	"fmt"
	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"
	"net/http"
	"runtime"
	"strings"
)

type filter struct {
	value string
	regex bool
	exclude bool
}

type rssHandler struct {
	apiKey string
}

func newRssHandler(apiKey string) *rssHandler {
	return &rssHandler{
		apiKey: apiKey,
	}
}

func (h rssHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if h.apiKey != "" && h.apiKey != r.Header.Get("x-api-key") {
		w.Header().Add("WWW-Authenticate", "API key is missing or invalid")
		w.WriteHeader(http.StatusUnauthorized)
		log.Warn().Str("key", r.Header.Get("c-api-key")).Msg("API key is missing or invalid")
		return
	}

	var feedUrl string

	q := r.URL.Query()
	l := log.With()
	for k, v := range q {
		if strings.ToLower(k) == "feed_url" && len(v) > 0 {
			feedUrl = v[0]
		} else if strings.ToLower(k) == "filter" {
			for _, s := range v {
				fmt.Println("FILTER", s)

			}
		}


		l = l.Strs(k, v)
	}
	ll := l.Logger()
	ll.Trace().Msg("serve http")

	fp := gofeed.NewParser()
	fp.UserAgent = userAgent()
	feed, err := fp.ParseURL(feedUrl)
	if err != nil {
		log.Err(err).Msg("parsing of feed failed")
		_, _ = w.Write([]byte(fmt.Sprintf("can't parse feed: %s", feedUrl)))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	items := feed.Items
	for i := 0; i < len(items); i++ {

	}

	log.Trace().Str("title", feed.Title).Send()
}

func userAgent() string {
	return fmt.Sprintf("rss-filter/%s (%s; %s)", version, runtime.GOOS, runtime.GOARCH)
}
