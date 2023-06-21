package sitemap

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	xurl "net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

type URL struct {
	Loc   string
	Level int
}

type Sitemap struct {
	IgnoreQuery    bool      // Ignore query string in URLs (i.e. example.com/?foo=bar)
	IgnoreFragment bool      // Ignore fragment in URLs (i.e. example.com/#fragment)
	ChangeFreq     string    // Add <changefreq> to URLs (not added if empty)
	LastMod        time.Time // Add custom <lastmod> to URLs (default is time.Now())
	Verbose        bool      // Verbose logging
	baseURL        *xurl.URL
	wg             sync.WaitGroup
	crawled        map[string]int
	crawledMutex   sync.Mutex
	httpClient     *http.Client
}

func New() *Sitemap {
	return &Sitemap{
		IgnoreQuery:    true,
		IgnoreFragment: true,
		Verbose:        false,
		baseURL:        nil,
		httpClient:     nil,
		wg:             sync.WaitGroup{},
		crawled:        nil,
		crawledMutex:   sync.Mutex{},
	}
}

// Will generate a XML Sitemap for the given URL and write it to the given writer.
func (s *Sitemap) Generate(w io.Writer, url *string) error {
	s.httpClient = &http.Client{
		Transport: &http.Transport{
			MaxIdleConnsPerHost: 20,
		},
		Timeout: 10 * time.Second,
	}
	s.crawled = make(map[string]int)

	s.parseBaseURL(url)

	baseUrl := s.baseURL.String()

	s.wg.Add(1)
	go s.crawlURL(&baseUrl, 0)
	s.wg.Wait()

	return s.writeXML(w)
}

func (s *Sitemap) mapCrawl(url *string, level int) bool {
	s.crawledMutex.Lock()
	defer s.crawledMutex.Unlock()

	_, ok := s.crawled[*url]
	if ok {
		if s.Verbose {
			log.Printf("Skipping %s due to already crawled", *url)
		}

		if level < s.crawled[*url] {
			s.crawled[*url] = level
		}
		return false
	}

	if s.Verbose {
		log.Printf("Crawling %s", *url)
	}

	s.crawled[*url] = level

	return true
}

func (s *Sitemap) invalidateURL(url *string) {
	s.crawledMutex.Lock()
	defer s.crawledMutex.Unlock()

	s.crawled[*url] = -1
}

func (s *Sitemap) crawlURL(rawURL *string, level int) {
	defer s.wg.Done()

	canCrawl := s.mapCrawl(rawURL, level)
	if !canCrawl {
		return
	}

	req, err := s.httpClient.Get((*rawURL)[0 : len(*rawURL)-1])
	if err != nil {
		if s.Verbose {
			log.Printf("Error crawling %s: %v", *rawURL, err)
		}

		s.invalidateURL(rawURL)
		return
	}
	defer req.Body.Close()

	if req.StatusCode != http.StatusOK {
		if s.Verbose {
			log.Printf("Skipping %s due to status code %d", *rawURL, req.StatusCode)
		}

		s.invalidateURL(rawURL)
		return
	}

	contentType := req.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "text/html") {
		if s.Verbose {
			log.Printf("Skipping %s due to content type %s", *rawURL, contentType)
		}

		s.invalidateURL(rawURL)
		return

	}

	doc, err := html.Parse(req.Body)
	if err != nil {
		if s.Verbose {
			log.Printf("Error parsing %s: %v", *rawURL, err)
		}
		return
	}

	s.searchNode(doc, level+1)
}

func (s *Sitemap) searchNode(n *html.Node, level int) {
	if n.Type == html.ElementNode && n.Data == "a" {
		for _, a := range n.Attr {
			if a.Key == "href" {
				rawURL := strings.TrimSpace(a.Val)
				url, err := s.parseURL(&rawURL)
				if err != nil {
					if s.Verbose {
						log.Printf("Error parsing URL %s: %v", a.Val, err)
					}
					continue
				}

				s.wg.Add(1)
				go s.crawlURL(url, level)
			}
		}
	}

	for c := n.FirstChild; c != nil; c = c.NextSibling {
		s.searchNode(c, level)
	}
}

func (s *Sitemap) parseBaseURL(rawURL *string) error {
	baseURL, err := xurl.Parse(*rawURL)
	if err != nil {
		return err
	}

	if baseURL.Scheme == "" {
		baseURL.Scheme = "http"
	}

	if baseURL.Host == "" {
		baseURL.Host = "localhost"
	}

	if baseURL.Path == "" {
		baseURL.Path = "/"
	}

	if baseURL.Path[len(baseURL.Path)-1] != '/' {
		baseURL.Path += "/"
	}

	if s.IgnoreQuery {
		baseURL.RawQuery = ""
	}

	s.baseURL = baseURL

	return nil
}

func (s *Sitemap) parseURL(rawURL *string) (*string, error) {
	u, err := xurl.Parse(*rawURL)
	if err != nil {
		return nil, err
	}

	if u.IsAbs() {
		if u.Scheme != s.baseURL.Scheme || u.Hostname() != s.baseURL.Hostname() || u.Port() != s.baseURL.Port() {
			return nil, fmt.Errorf("URL %s is not on the same domain as %s", u.String(), s.baseURL.String())
		}

	} else {
		u = s.baseURL.ResolveReference(u)
	}

	if s.IgnoreQuery {
		u.RawQuery = ""
	}

	if s.IgnoreFragment {
		u.Fragment = ""
	}

	if u.Path == "" {
		u.Path = "/"
	}

	if u.Path[len(u.Path)-1] != '/' {
		u.Path += "/"
	}

	url := u.String()

	return &url, nil
}

func (s *Sitemap) writeXML(w io.Writer) error {
	_, err := w.Write([]byte(`<?xml version="1.0" encoding="UTF-8"?>` + "\n" +
		`<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">` + "\n"),
	)
	if err != nil {
		return err
	}

	lastMod := s.LastMod
	if lastMod.IsZero() {
		lastMod = time.Now()
	}

	changeFreq := ""
	if s.ChangeFreq != "" {
		changeFreq = fmt.Sprintf("\n    <changefreq>%s</changefreq>", s.ChangeFreq)
	}

	urls := make([]*URL, 0)
	for url := range s.crawled {
		if s.crawled[url] == -1 {
			continue
		}

		urls = append(urls, &URL{
			Loc:   url,
			Level: s.crawled[url],
		})
	}

	sort.Slice(urls, func(i, j int) bool {
		if urls[i].Level == urls[j].Level {
			return urls[i].Loc < urls[j].Loc
		}

		return urls[i].Level < urls[j].Level
	})

	for i := range urls {
		url := urls[i]

		priority := math.Max(0.1, 1.0-(0.1*float64(url.Level)))

		_, err = w.Write([]byte(fmt.Sprintf("  <url>\n    <loc>%s</loc>\n    <lastmod>%s</lastmod>\n    <priority>%.1f</priority>%s\n  </url>\n",
			url.Loc,
			lastMod.Format(time.RFC3339),
			priority,
			changeFreq,
		)))
		if err != nil {
			return err
		}
	}

	_, err = w.Write([]byte(`</urlset>`))
	if err != nil {
		return err
	}

	return nil
}
