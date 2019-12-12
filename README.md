# spotlight-gel

> gel, is a transparent material that is used in theater, event production,
> photography, videography and cinematography to color light and for color
> correction.  ([wikipedia](https://en.wikipedia.org/wiki/Color_gel)

## What does it do

This is a hacked up fork of
[go-sitemirror](https://godoc.org/github.com/daohoangson/go-sitemirror/) that
is used as a temporary way to serve a static/sanitised mirror of
[spotlight](https://github.com/alphagov/spotlight/) while it remains behind an
internal network.


* Crawls spotlight on it's internal address and warms a cache of it's content
* onto disk Automatically refreshes the cache periodically (every few hours)
* Exposes a http server that presents a mirror of spotlight on the original
* URLs serving only from the cached data Strips out `script` tags from content
* to disable js

## Deploying

This application is intended to steal the public route from spotlight and be
pushed to GOV.UK PaaS alongside spotlight and pointing at spotlight's internal
app domain.

So first ensure that spotlight is configured with an `apps.internal` domain,
and does not have a public route.

Then add a network-policy so that "gel" can talk to "spotlight"...

```
cf add-network-policy performance-platform-spotlight-gel-staging \
	--destination-app performance-platform-spotlight-staging \
	--protocol tcp \
	--port 8080
```

Then push the app...

```
cf push -f manifest.staging.yaml
```

Wait for ages (takes about an hour to "warm up" the cache).

Then check it's serving...

```
curl http://performance-platform-spotlight-staging.cloudapps.digital
```

