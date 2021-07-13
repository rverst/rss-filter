package main

import (
	"fmt"
	"github.com/gorilla/feeds"
	"github.com/mmcdole/gofeed"
	"github.com/rs/zerolog/log"
	"github.com/rverst/goql"
	"io/ioutil"
	"net/http"
	"runtime"
	"strings"
	"time"
)

type format string

const (
	keep = format("keep")
	rss  = format("rss")
	atom = format("atom")
	json = format("json")
)

type rssHandler struct {
	user        string
	password    string
	disableAuth bool
}

func newRssHandler(user, password string, disableAuth bool) *rssHandler {
	return &rssHandler{
		user:        user,
		password:    password,
		disableAuth: disableAuth,
	}
}

func (h rssHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if !h.disableAuth {
		user, pass, ok := r.BasicAuth()
		if !ok || user != h.user || pass != h.password {
			w.Header().Add("WWW-Authenticate", "Basic realm=\"Access to rss-filter\", charset=\"UTF-8\"")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
	}

	var feedUrl, filter, output string

	q := r.URL.Query()
	for k, v := range q {
		if strings.ToLower(k) == "feed_url" && len(v) > 0 {
			feedUrl = v[0]
		} else if strings.ToLower(k) == "filter" && len(v) > 0 {
			filter = v[0]
		} else if strings.ToLower(k) == "out" && len(v) > 0 {
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

	req, err := http.NewRequest("GET", feedUrl, nil)
	if err != nil {
		log.Err(err).Msg("fetching of feed failed")
		w.WriteHeader(http.StatusInternalServerError)
	}

	req.Header.Set("User-Agent", userAgent())
	fUser := r.Header.Get("x-forward-user")
	fPass := r.Header.Get("x-forward-password")
	if fUser != "" || fPass != "" {
		req.SetBasicAuth(fUser, fPass)
	}

	client := http.DefaultClient
	resp, err := client.Do(req)

	if err != nil {
		log.Err(err).Msg("fetching of feed failed")
		w.WriteHeader(http.StatusInternalServerError)
	}

	if resp != nil {
		defer resp.Body.Close()
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Error().Int("status_code", resp.StatusCode).Str("status", resp.Status).Msg("http error")

		w.WriteHeader(resp.StatusCode)
		if data, err := ioutil.ReadAll(resp.Body); err != nil {
			log.Err(err).Send()
		} else {
			_, _ = w.Write(data)
		}
		return
	}

	fp := gofeed.NewParser()
	feed, err := fp.Parse(resp.Body)
	if err != nil {
		log.Err(err).Msg("parsing of feed failed")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("can't parse feed: %s", feedUrl)))
		return
	}

	if fm == keep {
		switch format(strings.ToLower(feed.FeedType)) {
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
		Author: &feeds.Author{
			Name: "https://github.com/rverst/rss-filter",
		},
		Updated:   *feed.UpdatedParsed,
		Created:   *feed.PublishedParsed,
		Items:     []*feeds.Item{},
		Copyright: feed.Copyright,
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
			Enclosure:   enc,
		})
	}

	var body string
	var cType = "application/xml"
	switch fm {
	case rss:
		body, err = newFeed.ToRss()
	case atom:
		body, err = newFeed.ToAtom()
	case json:
		body, err = newFeed.ToJSON()
		cType = "application/json"
	}
	if err != nil {
		log.Err(err).Msg("creating of feed failed")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(fmt.Sprintf("can't create feed: %#v", newFeed)))
		return
	}
	log.Debug().Str("format", string(fm)).Int("original_items", len(feed.Items)).Int("kept_items", len(newFeed.Items)).Msg("feed filtered")

	w.Header().Set("Content-Type", cType)
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(body))
}

func userAgent() string {
	return fmt.Sprintf("rss-filter/%s (%s; %s)", version, runtime.GOOS, runtime.GOARCH)
}
