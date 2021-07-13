# rss-filter

The purpose of the rss filter is to filter out certain parts from a feed (rss/atom). I have built 
the filter because the daily newspaper, whose feed I have subscribed to, has turned off its 
individual regional feeds and replaced them with a common one.

### Usage:

docker:
```
> docker pull ghcr.io/rverst/rss-filter:latest
> docker run -p 80:80 -e PASSWORD=secret ghcr.io/rverst/rss-filte:latest
```

Now you can make requests to the rss-filter and get a filtered feed in response.

### Environment variables

| variable | meaning |
|----------|---------|
| LISTEN_ADDR   | The address the container should listen on (address:port, default :80) |
| AUTH_USER     | The username used for basic http authentication of the endpoint |
| AUTH_PASSWORD | The password used for basic http authentication of the endpoint |
| DISABLE_AUTH  | Disable the authentication for the endpoint (boolean) |

### URL parameters:

The filter is controlled by url parameter:

| parameter | meaning |
|-----------|---------|
| feed_url  | address of the feed to be retrieved |
| filter    | filter to be applied, e.g. ` Title ~= "^Breaking.*"` |
| out       | output format of the feed (rss/atom/json/keep), `keep` is default, the original format is used. |

### Headers

You can provide the headers `x-forward-user`, `x-forward-password` to the request. 
These values are then used to perform basic authentication to the feed server.

| header | meaning |
|--------|---------|
| x-forward-user | the `user` part of a basic http authentication |
| x-forward-password | the `password` part of a basic http authentication |


### Filtering

The filter provided in the url parameter is parsed with [goql](https://github.com/rverst/goql)
and then applied on the
[Item struct of github.com/mmcdole/gofeed parser](https://github.com/mmcdole/gofeed/blob/41f47c9aa28b0731e0ac1b5a92830b1951ba91c9/feed.go#L49).

For now the filter can be applied to all simple fields (string,int,bool etc. and time.Time) 
of the structure. For example:

`Title != "Foo Bar"` -> `Title` must not be "Foo Bar".

Most useful for this use case (at least for mine) is
probably the regex filter:
- `~=` - regex must match
- `~!` - regex must not match

e.g. `Link ~= "^https://example.org/category/a.*"` -> link must start with `https://..`.

Several filters can also be linked with AND (&) or OR (|).

`Link ~= "^https://example.org" & Title ~! "^Breaking"`


> You probably want to use an online service to encode the URL parameters ;-)
