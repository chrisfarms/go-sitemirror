package crawler_test

import (
	"fmt"
	"net/http"
	neturl "net/url"

	"gopkg.in/jarcoal/httpmock.v1"

	. "github.com/daohoangson/go-sitemirror/crawler"
	t "github.com/daohoangson/go-sitemirror/testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Download", func() {
	BeforeEach(func() {
		httpmock.Activate()
	})

	AfterEach(func() {
		httpmock.DeactivateAndReset()
	})

	It("should not work with nil http.Client", func() {
		url := "http://domain.com/client/nil"

		parsedURL, _ := neturl.Parse(url)
		Expect(parsedURL).ToNot(BeNil())
		downloaded := Download(nil, parsedURL)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should not work with nil url.URL", func() {
		downloaded := Download(http.DefaultClient, nil)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should not work with relative url", func() {
		url := "relative/url/"
		parsedURL, _ := neturl.Parse(url)

		Expect(parsedURL).ToNot(BeNil())
		downloaded := Download(http.DefaultClient, parsedURL)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should not work with non http/https url", func() {
		url := "ftp://domain.com/non/http/url"
		parsedURL, _ := neturl.Parse(url)

		Expect(parsedURL).ToNot(BeNil())
		downloaded := Download(http.DefaultClient, parsedURL)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	It("should passthrough client error", func() {
		url := "http://a.b.c"
		parsedURL, _ := neturl.Parse(url)

		Expect(parsedURL).ToNot(BeNil())
		downloaded := Download(http.DefaultClient, parsedURL)

		Expect(downloaded.Error).To(HaveOccurred())
	})

	Describe("BaseURL", func() {
		It("should match url", func() {
			url := "http://domain.com/download/url/base"
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(""))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BaseURL).To(Equal(parsedURL))
			Expect(downloaded.URL).To(Equal(parsedURL))
		})

		It("should match base href", func() {
			url := "http://domain.com/download/url/base/href"
			baseHref := "/some/where/else"
			htmlTemplate := "<base href=\"%s\" />"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, baseHref))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "."))))
			Expect(downloaded.BaseURL.String()).To(Equal("http://domain.com/some/where/else"))
			Expect(downloaded.URL).To(Equal(parsedURL))
		})

		It("should match url on empty base href", func() {
			url := "http://domain.com/download/url/base/href/empty"
			html := t.NewHtmlMarkup("<base />")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(downloaded.BaseURL).To(Equal(parsedURL))
			Expect(downloaded.URL).To(Equal(parsedURL))
		})
	})

	Describe("Body", func() {
		It("should match generic response body", func() {
			url := "http://domain.com/download/body"
			body := "foo/bar"
			httpmock.RegisterResponder("GET", url, httpmock.NewStringResponder(200, body))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(len(downloaded.BodyString)).To(Equal(0))
			Expect(string(downloaded.BodyBytes)).To(Equal(body))
		})

		It("should match css", func() {
			url := "http://domain.com/download/body/css/valid"
			css := "body{background:#fff}"
			httpmock.RegisterResponder("GET", url, t.NewCssResponder(css))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(css))
		})

		It("should match valid html", func() {
			url := "http://domain.com/download/body/html/valid"
			html := t.NewHtmlMarkup("<p>Hello World!</p>")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
		})

		It("should keep invalid html intact", func() {
			url := "http://domain.com/download/body/html/invalid"
			html := t.NewHtmlMarkup("<p>Oops</p")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
		})
	})

	Describe("ContentType", func() {
		It("should match response header value", func() {
			url := "http://domain.com/download/content/type"
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(""))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.ContentType).To(Equal("text/html"))
		})
	})

	Describe("HeaderLocation", func() {
		It("should work with 301 response status", func() {
			status := 301
			url := fmt.Sprintf("http://domain.com/download/header/location/%d", status)
			targetUrl := "http://domain.com/download/target/url"
			httpmock.RegisterResponder("GET", url, t.NewRedirectResponder(status, targetUrl))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.StatusCode).To(Equal(status))
			Expect(downloaded.HeaderLocation.String()).To(Equal(targetUrl))
		})

		It("should not work with invalid url", func() {
			// have to use 399 status code otherwise http.Client will parse
			// the location header itself and trigger error too soon
			status := 399
			url := "http://domain.com/download/header/location/invalid"
			targetUrl := t.InvalidURL
			httpmock.RegisterResponder("GET", url, t.NewRedirectResponder(status, targetUrl))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.StatusCode).To(Equal(status))
			Expect(downloaded.HeaderLocation).To(BeNil())
		})
	})

	Describe("Links", func() {
		It("should pick up css url() value", func() {
			url := "http://domain.com/download/urls/css/url"
			targetUrl := "http://domain.com/download/urls/css/target"
			cssTemplate := "body{background:url('%s')}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			httpmock.RegisterResponder("GET", url, t.NewCssResponder(css))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(fmt.Sprintf(cssTemplate, "target")))
			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrl))
			Expect(downloaded.Links[0].Context).To(Equal(CSSUri))
		})

		It("should pick up css url() value, double quote", func() {
			url := "http://domain.com/download/urls/css/url/double/quote"
			targetUrl := "http://domain.com/download/urls/css/url/double/target"
			cssTemplate := "body{background:url(\"%s\")}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			httpmock.RegisterResponder("GET", url, t.NewCssResponder(css))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(fmt.Sprintf(cssTemplate, "target")))
			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrl))
			Expect(downloaded.Links[0].Context).To(Equal(CSSUri))
		})

		It("should pick up css url() value, no quote", func() {
			url := "http://domain.com/download/urls/css/url/no/quote"
			targetUrl := "http://domain.com/download/urls/css/url/no/target"
			cssTemplate := "body{background:url(%s)}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			httpmock.RegisterResponder("GET", url, t.NewCssResponder(css))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(fmt.Sprintf(cssTemplate, "target")))
			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrl))
			Expect(downloaded.Links[0].Context).To(Equal(CSSUri))
		})

		It("should pick up a href", func() {
			url := "http://domain.com/download/urls/a"
			targetUrl := "http://domain.com/download/urls/target"
			htmlTemplate := "<a href=\"%s\">Link</a><a>Anchor</a>"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "target"))))
			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrl))
			Expect(downloaded.Links[0].Context).To(Equal(HTMLTagA))
		})

		It("should pick up script src", func() {
			url := "http://domain.com/download/urls/script"
			targetUrl := "http://domain.com/download/urls/target"
			htmlTemplate := "<script src=\"%s\"></script><script>alert('hello world');</script>"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "target"))))
			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrl))
			Expect(downloaded.Links[0].Context).To(Equal(HTMLTagScript))
		})

		It("should pick up inline css url() value", func() {
			url := "http://domain.com/download/urls/inline/css/url"
			targetUrl := "http://domain.com/download/urls/inline/css/target"
			cssTemplate := "body{background:url('%s')}"
			css := fmt.Sprintf(cssTemplate, targetUrl)
			htmlTemplate := "<style>%s</style>"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, css))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			cssNew := fmt.Sprintf(cssTemplate, "target")
			htmlNew := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, cssNew))
			Expect(downloaded.BodyString).To(Equal(htmlNew))
			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrl))
			Expect(downloaded.Links[0].Context).To(Equal(CSSUri))
		})

		It("should pick up img src", func() {
			url := "http://domain.com/download/urls/img"
			targetUrl := "http://domain.com/download/urls/target"
			htmlTemplate := "<img src=\"%s\" /><img />"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "target"))))
			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrl))
			Expect(downloaded.Links[0].Context).To(Equal(HTMLTagImg))
		})

		It("should pick up link[rel=stylesheet] href", func() {
			url := "http://domain.com/download/urls/link/stylesheet"
			targetUrl := "http://domain.com/download/urls/link/target"
			htmlTemplate := "<link rel=\"stylesheet\" href=\"%s\" /><link />"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "target"))))
			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrl))
			Expect(downloaded.Links[0].Context).To(Equal(HTMLTagLinkStylesheet))
		})

		It("should pick up 3xx response Location header", func() {
			url := "http://domain.com/download/urls/3xx"
			targetUrl := "http://domain.com/download/target/url"
			httpmock.RegisterResponder("GET", url, t.NewRedirectResponder(301, targetUrl))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrl))
			Expect(downloaded.Links[0].Context).To(Equal(HTTP3xxLocation))
		})

		It("should pick up multiple urls", func() {
			url := "http://domain.com/download/urls/multiple"
			targetUrlA := "http://domain.com/download/target/url/a"
			targetUrlAHttps := "https://domain.com/download/target/url/a/https"
			targetUrlAProtocolRelative := "//domain.com/download/target/url/a/protocol/relative"
			targetUrlScript := "http://domain.com/download/target/url/script"
			targetUrlCssUri := "http://domain.com/download/target/url/css/uri"
			targetUrlImg := "http://domain.com/download/target/url/img"
			targetUrlLink := "http://domain.com/download/target/url/link"
			css := fmt.Sprintf("body{background:url('%s')}", targetUrlCssUri)
			html := t.NewHtmlMarkup(
				fmt.Sprintf("<a href=\"%s\">Link</a>", targetUrlA) +
					fmt.Sprintf("<a href=\"%s\">Link HTTPS</a>", targetUrlAHttps) +
					fmt.Sprintf("<a href=\"%s\">Link protocol relative</a>", targetUrlAProtocolRelative) +
					fmt.Sprintf("<script src=\"%s\"></script>", targetUrlScript) +
					fmt.Sprintf("<style>%s</style>", css) +
					fmt.Sprintf("<img src=\"%s\" />", targetUrlImg) +
					fmt.Sprintf("<link rel=\"stylesheet\" href=\"%s\" />", targetUrlLink))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(len(downloaded.Links)).To(Equal(7))

			urls := downloaded.Links
			Expect(urls[0].URL.String()).To(Equal(targetUrlA))
			Expect(urls[0].Context).To(Equal(HTMLTagA))

			urls = urls[1:]
			Expect(urls[0].URL.String()).To(Equal(targetUrlAHttps))
			Expect(urls[0].Context).To(Equal(HTMLTagA))

			urls = urls[1:]
			Expect(urls[0].URL.String()).To(Equal("http:" + targetUrlAProtocolRelative))
			Expect(urls[0].Context).To(Equal(HTMLTagA))

			urls = urls[1:]
			Expect(urls[0].URL.String()).To(Equal(targetUrlScript))
			Expect(urls[0].Context).To(Equal(HTMLTagScript))

			urls = urls[1:]
			Expect(urls[0].URL.String()).To(Equal(targetUrlCssUri))
			Expect(urls[0].Context).To(Equal(CSSUri))

			urls = urls[1:]
			Expect(urls[0].URL.String()).To(Equal(targetUrlImg))
			Expect(urls[0].Context).To(Equal(HTMLTagImg))

			urls = urls[1:]
			Expect(urls[0].URL.String()).To(Equal(targetUrlLink))
			Expect(urls[0].Context).To(Equal(HTMLTagLinkStylesheet))
		})

		It("should not pick up empty url", func() {
			url := "http://domain.com/download/urls/empty/url"
			html := t.NewHtmlMarkup("<a href=\"\">Link</a>")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.Links)).To(Equal(0))
		})

		It("should not pick up invalid url", func() {
			url := "http://domain.com/download/urls/invalid/url"
			html := t.NewHtmlMarkup(fmt.Sprintf("<a href=\"%s\">Link</a>", t.InvalidURL))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.Links)).To(Equal(0))
		})

		It("should not pick up non http/https url", func() {
			url := "http://domain.com/download/urls/non/http/url"
			html := t.NewHtmlMarkup("<a href=\"ftp://domain.com/non/http/url\">Link</a>")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.Links)).To(Equal(0))
		})

		It("should not pick up data uri", func() {
			url := "http://domain.com/download/urls/data/uri"
			html := t.NewHtmlMarkup(fmt.Sprintf("<img src=\"%s\" />", t.TransparentDataURI))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.Links)).To(Equal(0))
		})

		It("should not pick up url #fragment part", func() {
			url := "http://domain.com/download/urls/fragment"
			targetUrlBase := "http://domain.com/download/urls/target"
			fragment := "#foo=bar"
			targetUrl := targetUrlBase + fragment
			htmlTemplate := "<a href=\"%s\">Link</a>"
			html := t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, targetUrl))
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(t.NewHtmlMarkup(fmt.Sprintf(htmlTemplate, "target"+fragment))))
			Expect(len(downloaded.Links)).To(Equal(1))
			Expect(downloaded.Links[0].URL.String()).To(Equal(targetUrlBase))
		})

		It("should not pick up #fragment only url", func() {
			url := "http://domain.com/download/urls/fragment/only"
			html := t.NewHtmlMarkup("<a href=\"#\">Link</a>")
			httpmock.RegisterResponder("GET", url, t.NewHtmlResponder(html))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.BodyString).To(Equal(html))
			Expect(len(downloaded.Links)).To(Equal(0))
		})
	})

	Describe("StatusCode", func() {
		It("should match response status code", func() {
			url := "http://domain.com/download/status/code"
			statusCode := 200
			httpmock.RegisterResponder("GET", url,
				httpmock.NewStringResponder(statusCode, ""))

			parsedURL, _ := neturl.Parse(url)
			Expect(parsedURL).ToNot(BeNil())
			downloaded := Download(http.DefaultClient, parsedURL)

			Expect(downloaded.StatusCode).To(Equal(statusCode))
		})
	})

	Describe("Downloaded", func() {
		baseUrl, _ := neturl.Parse("http://domain.com/downloaded/base")
		var downloaded *Downloaded

		BeforeEach(func() {
			downloaded = &Downloaded{
				BaseURL: baseUrl,
				Links:   make([]Link, 0),
			}
		})

		Describe("GetResolvedURL", func() {
			It("should not resolve invalid index", func() {
				resolved := downloaded.GetResolvedURL(0)
				Expect(resolved).To(BeNil())
			})

			It("should resolve relative url", func() {
				linkUrl, _ := neturl.Parse("relative")
				link := Link{URL: linkUrl}
				downloaded.Links = append(downloaded.Links, link)

				resolved := downloaded.GetResolvedURL(0)
				Expect(resolved.String()).To(Equal("http://domain.com/downloaded/relative"))
			})

			It("should resolve root relative url", func() {
				linkUrl, _ := neturl.Parse("/root/relative")
				link := Link{URL: linkUrl}
				downloaded.Links = append(downloaded.Links, link)

				resolved := downloaded.GetResolvedURL(0)
				Expect(resolved.String()).To(Equal("http://domain.com/root/relative"))
			})

			It("should keep absolute url intact", func() {
				linkUrl, _ := neturl.Parse("http://domain2.com")
				link := Link{URL: linkUrl}
				downloaded.Links = append(downloaded.Links, link)

				resolved := downloaded.GetResolvedURL(0)
				Expect(resolved.String()).To(Equal(linkUrl.String()))
			})
		})
	})
})
