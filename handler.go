package main

import (
	"fmt"
	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"
	"github.com/rverst/goql"
	"net/http"
	"runtime"
	"strings"
	"time"
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

	var feedUrl, filter string

	q := r.URL.Query()
	l := log.With()
	for k, v := range q {
		if strings.ToLower(k) == "feed_url" && len(v) > 0 {
			feedUrl = v[0]
		} else if strings.ToLower(k) == "filter" && len(v) > 0{
			filter = v[0]
		}

		l = l.Strs(k, v)
	}
	ll := l.Logger()
	ll.Trace().Msg("serve http")

	p := goql.NewParser(strings.NewReader(filter))
	t, err := p.Parse()
	if err != nil {
		log.Err(err).Msg("parsing filter failed")
		_, _ = w.Write([]byte(fmt.Sprintf("can't parse filter: %s", err.Error())))
		w.WriteHeader(http.StatusInternalServerError)
	}

	fp := gofeed.NewParser()
	fp.UserAgent = userAgent()
	feed, err := fp.ParseURL(feedUrl)
	if err != nil {
		log.Err(err).Msg("parsing of feed failed")
		_, _ = w.Write([]byte(fmt.Sprintf("can't parse feed: %s", feedUrl)))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newFeed := feeds.Feed{
		Title:       feed.Title,
		Link:        &feeds.Link{Href: feed.Link},
		Description: feed.Description,
		Updated:     *feed.UpdatedParsed,
		Created:     *feed.PublishedParsed,
		Items: 		 []*feeds.Item{},
		Copyright:   feed.Copyright,
	}

	for _, item := range feed.Items {
		if item == nil {
			continue
		}
		b, err := t.CheckStruct(item)
		if err != nil {
			log.Warn().Err(err).Interface("item", item).Msg("check item failed")
		}
		if !b {
			continue
		}

		var pub, upd time.Time
		var enc *feeds.Enclosure
		if item.PublishedParsed != nil {
			pub = *item.PublishedParsed
		}
		if item.UpdatedParsed != nil {
			upd = *item.UpdatedParsed
		}

		if len(item.Enclosures) > 0 {
			enc = new(feeds.Enclosure)
			enc.Type = item.Enclosures[0].Type
			enc.Url = item.Enclosures[0].URL
			enc.Length = item.Enclosures[0].Length
		}

		newFeed.Items = append(newFeed.Items, &feeds.Item{
			Title:       item.Title,
			Link:        &feeds.Link{Href: item.Link},
			Description: item.Description,
			Id:          item.GUID,
			Updated:     upd,
			Created:     pub,
			Content:     item.Content,
			Enclosure:  enc,
		})
	}

	json, err := newFeed.ToJSON()
	if err != nil {
		log.Err(err).Msg("creating of feed failed")
		_, _ = w.Write([]byte(fmt.Sprintf("can't create feed: %#v", newFeed)))
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(json))
	w.WriteHeader(http.StatusOK)
}

func userAgent() string {
	return fmt.Sprintf("rss-filter/%s (%s; %s)", version, runtime.GOOS, runtime.GOARCH)
}
