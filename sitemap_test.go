package sitemap

import (
	"bytes"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestSitemap(t *testing.T) {
	server, err := createServer()
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
		return
	}

	url := "http://" + server.Addr
	go func() {
		err = server.ListenAndServe()
		if err != nil {
			if err == http.ErrServerClosed {
				t.Logf("Server closed")
				return
			} else {
				t.Errorf("Error starting server: %v", err)
			}
		}
	}()

	tries := 0
	for {
		t.Log("Waiting for server to start...")

		time.Sleep(100 * time.Millisecond)

		_, err := http.Get(url)
		if err == nil {
			break
		}

		tries++
		if tries > 5 {
			t.Errorf("Expected server to start, got %v", err)
			return
		}
	}

	t.Run("TestDefault", func(t *testing.T) {
		t.Parallel()
		var writer bytes.Buffer
		sitemap := New()

		err := sitemap.Generate(&writer, &url)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		expected, err := os.ReadFile("fixtures/sitemap_1.xml")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		if writer.String() != string(expected) {
			t.Errorf("Expected %v, got %v", string(expected), writer.String())
			return
		}
	})

	t.Run("TestQueryChangeFreqLastMod", func(t *testing.T) {
		t.Parallel()
		var writer bytes.Buffer
		sitemap := New()
		sitemap.IgnoreQuery = false
		sitemap.ChangeFreq = "monthly"
		sitemap.LastMod = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		err := sitemap.Generate(&writer, &url)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		expected, err := os.ReadFile("fixtures/sitemap_2.xml")
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
			return
		}

		if writer.String() != string(expected) {
			t.Errorf("Expected %v, got %v", string(expected), writer.String())
			return
		}
	})
}

func createServer() (*http.Server, error) {
	mux := http.NewServeMux()

	home, err := os.ReadFile("fixtures/site/index.html")
	if err != nil {
		return nil, err
	}
	tac, err := os.ReadFile("fixtures/site/terms-and-conditions.html")
	if err != nil {
		return nil, err
	}
	aboutus, err := os.ReadFile("fixtures/site/about-us.html")
	if err != nil {
		return nil, err
	}
	relative, err := os.ReadFile("fixtures/site/relative.html")
	if err != nil {
		return nil, err
	}

	mux.Handle("/levels/", http.StripPrefix("/levels", http.FileServer(http.Dir("fixtures/site/levels"))))
	mux.HandleFunc("/terms-and-conditions", func(w http.ResponseWriter, r *http.Request) {
		w.Write(tac)
	})
	mux.HandleFunc("/about-us", func(w http.ResponseWriter, r *http.Request) {
		w.Write(aboutus)
	})
	mux.HandleFunc("/relative", func(w http.ResponseWriter, r *http.Request) {
		w.Write(relative)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "" || r.URL.Path == "/" {
			w.Write(home)
			return
		}

		http.NotFound(w, r)
	})

	return &http.Server{
		Addr:    "localhost:9876",
		Handler: mux,
	}, nil
}
