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

type format string

const (
	keep = format("keep")
	rss = format("rss")
	atom = format("atom")
	json = format("json")
)

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

	var feedUrl, filter, output string

	q := r.URL.Query()
	for k, v := range q {
		if strings.ToLower(k) == "feed_url" && len(v) > 0 {
			feedUrl = v[0]
		} else if strings.ToLower(k) == "filter" && len(v) > 0{
			filter = v[0]
		} else if strings.ToLower(k) == "out" && len(v) > 0{
			output = strings.ToLower(v[0])
		}
	}
	log.Trace().Str("feed_url", feedUrl).Str("filter", filter).Str("output", output).Msg("serve http")

	if feedUrl == "" {
		log.Error().Msg("no feed provided")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("no feed url"))
		return
	}

	var fm format
	switch format(output) {
	case rss:
		fm = rss
	case atom:
		fm = atom
	case json:
		fm = json
	default:
		fm = keep
	}

	p := goql.NewParser(strings.NewReader(filter))
	t, err := p.Parse()
	if err != nil {
		log.Err(err).Msg("parsing filter failed")
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(fmt.Sprintf("can't parse filter: %s", err.Error())))
		return
	}

	fp := gofeed.NewParser()
	fp.UserAgent = userAgent()
	feed, err := fp.ParseURL(feedUrl)
	if err != nil {
		log.Err(err).Msg("parsing of feed failed")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("can't parse feed: %s", feedUrl)))
		return
	}

	if fm == keep {
		switch format(strings.ToLower(feed.FeedType)){
		case rss:
			fm = rss
		case atom:
			fm = atom
		case json:
			fm = json
		default:
			fm = atom
		}
	}

	newFeed := feeds.Feed{
		Title:       feed.Title,
		Link:        &feeds.Link{Href: feed.Link},
		Description: feed.Description,
		Author:      &feeds.Author {
			Name: "https://github.com/rverst/rss-filter",
		},
		Updated:     *feed.UpdatedParsed,
		Created:     *feed.PublishedParsed,
		Items:       []*feeds.Item{},
		Copyright:   feed.Copyright,
	}

	for _, item := range feed.Items {
		if item == nil {
			continue
		}
		if len(t.Conditions()) > 0 {
			b, err := t.CheckStruct(item)
			if err != nil {
				log.Warn().Err(err).Interface("item", item).Msg("check item failed")
			}
			if !b {
				continue
			}
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

	var body string
	var ctype = "application/xml"
	switch fm {
	case rss:
		body, err = newFeed.ToRss()
	case atom:
		body, err = newFeed.ToAtom()
	case json:
		body, err = newFeed.ToJSON()
		ctype = "application/json"
	}
	if err != nil {
		log.Err(err).Msg("creating of feed failed")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("can't create feed: %#v", newFeed)))
		return
	}
	log.Debug().Str("format", string(fm)).Int("original_items", len(feed.Items)).Int("kept_items", len(newFeed.Items)).Msg("feed filtered")

	w.Header().Set("Content-Type", ctype)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func userAgent() string {
	return fmt.Sprintf("rss-filter/%s (%s; %s)", version, runtime.GOOS, runtime.GOARCH)
}
