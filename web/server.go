package web

import (
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/daohoangson/go-sitemirror/cacher"
	"github.com/daohoangson/go-sitemirror/web/internal"
)

type server struct {
	cacher cacher.Cacher
	logger *logrus.Logger

	onServerIssue *func(*ServerIssue)

	listeners map[string]net.Listener
}

var (
	regexpCrossHostPath = regexp.MustCompile(`^/(https?)/([^/]+)(/.*)?$`)
)

// NewServer returns a new server intance
func NewServer(cacher cacher.Cacher, logger *logrus.Logger) Server {
	s := &server{}
	s.init(cacher, logger)
	return s
}

func (s *server) init(httpCacher cacher.Cacher, logger *logrus.Logger) {
	if httpCacher == nil {
		httpCacher = cacher.NewHTTPCacher(nil)
	}
	s.cacher = httpCacher

	if logger == nil {
		logger = logrus.New()
	}
	s.logger = logger

	s.listeners = make(map[string]net.Listener)
}

func (s *server) GetCacher() cacher.Cacher {
	return s.cacher
}

func (s *server) SetOnServerIssue(f func(*ServerIssue)) {
	s.onServerIssue = &f
}

func (s *server) ListenAndServe(root *url.URL, port int) (io.Closer, error) {
	if port < 0 {
		return nil, errors.New("Invalid port")
	}

	var host string
	if root != nil {
		host = root.Host
	}
	if existing, existingFound := s.listeners[host]; existingFound {
		return existing, errors.New("Existing listener has been found for this host")
	}

	loggerContext := s.logger.WithFields(logrus.Fields{
		"root": root,
		"port": port,
	})

	listener, listenError := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if listenError != nil {
		loggerContext.WithField("error", listenError).Errorf("Cannot listen")
		return nil, listenError
	}

	if port == 0 {
		loggerContext = loggerContext.WithField("addr", listener.Addr().String())
	}

	start := time.Now()
	go func() {
		var f http.HandlerFunc = func(w http.ResponseWriter, req *http.Request) {
			s.Serve(root, w, req)
		}

		loggerContext.Info("Serving")
		serveError := http.Serve(listener, f)
		if serveError != nil {
			elapsed := time.Since(start)
			errorContext := loggerContext.WithField("error", serveError)
			if elapsed > 50*time.Millisecond {
				// some time has passed, it's likely that it worked
				// but the listener has been asked to be closed
				errorContext.Debug("Listener has been closed")
			} else {
				errorContext.Errorf("Cannot serve")
			}
		}
	}()

	s.listeners[host] = listener
	return listener, nil
}

func (s *server) GetListeningPort(host string) (int, error) {
	listener, ok := s.listeners[host]
	if !ok {
		return 0, errors.New("Listener not found")
	}

	addr := listener.Addr().String()
	matches := regexp.MustCompile(`:(\d+)$`).FindStringSubmatch(addr)
	port, err := strconv.ParseInt(matches[1], 10, 64)

	return int(port), err
}

func (s *server) Serve(root *url.URL, w http.ResponseWriter, req *http.Request) internal.ServeInfo {
	si := internal.NewServeInfo(w)

	targetURL, _ := url.Parse(req.URL.String())
	if root != nil {
		targetURL.Scheme = root.Scheme
		targetURL.Host = root.Host
	} else {
		// cross-host serving (root=nil)
		matches := regexpCrossHostPath.FindStringSubmatch(targetURL.Path)
		if matches == nil {
			fmt.Printf("%s\n", targetURL.Path)
			return s.serveServerIssue(&ServerIssue{
				Type: CrossHostInvalidPath,
				URL:  targetURL,
				Info: si.OnCrossHostInvalidPath(),
			})
		}

		targetURL.Scheme = matches[1]
		targetURL.Host = matches[2]
		targetURL.Path = matches[3]
	}
	if len(targetURL.Scheme) == 0 {
		targetURL.Scheme = "http"
	}

	if len(req.Method) > 0 && req.Method != "GET" {
		return s.serveServerIssue(&ServerIssue{
			Type: MethodNotAllowed,
			URL:  targetURL,
			Info: si.OnMethodNotAllowed(),
		})
	}

	cache, err := s.cacher.Open(targetURL)
	if err != nil {
		return s.serveServerIssue(&ServerIssue{
			Type: CacheNotFound,
			URL:  targetURL,
			Info: si.OnCacheNotFound(err),
		})
	}
	defer cache.Close()

	ServeHTTPCache(cache, si)
	if si.HasError() {
		return s.serveServerIssue(&ServerIssue{
			Type: CacheError,
			URL:  targetURL,
			Info: si,
		})
	}

	loggerContext := s.logger.WithField("url", targetURL)
	siExpires := si.GetExpires()
	if siExpires != nil && siExpires.Before(time.Now()) {
		loggerContext = loggerContext.WithField("expired", siExpires)
		s.triggerOnServerIssue(&ServerIssue{
			Type: CacheExpired,
			URL:  targetURL,
			Info: si,
		})
	}

	loggerContext.WithField("statusCode", si.GetStatusCode()).Debug("Served")
	return si.Flush()
}

func (s *server) Stop() []string {
	hosts := make([]string, 0)

	for host, listener := range s.listeners {
		err := listener.Close()
		loggerContext := s.logger.WithField("host", host)

		if err == nil {
			loggerContext.Info("Stopped listening")
			hosts = append(hosts, host)
		} else {
			loggerContext.WithField("error", err).Error("Cannot stop listening")
		}

		delete(s.listeners, host)
	}

	return hosts
}

func (s *server) serveServerIssue(issue *ServerIssue) internal.ServeInfo {
	s.triggerOnServerIssue(issue)
	issue.Info.Flush()

	_, siError := issue.Info.GetError()
	s.logger.WithFields(logrus.Fields{
		"url":        issue.URL,
		"issue":      issue.Type,
		"error":      siError,
		"statusCode": issue.Info.GetStatusCode(),
	}).Debug("Served")

	return issue.Info
}

func (s *server) triggerOnServerIssue(issue *ServerIssue) {
	if s.onServerIssue == nil {
		return
	}

	(*s.onServerIssue)(issue)
}
