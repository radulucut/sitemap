# sitemap

**[XML Sitemap](https://www.sitemaps.org/protocol.html) generator**

![Test](https://github.com/radulucut/sitemap/actions/workflows/test.yml/badge.svg)

## Install

`go get github.com/radulucut/sitemap`

## Usage

```go
package main

import (
	"fmt"
	"os"

	"github.com/radulucut/sitemap"
)

func main() {
	file, err := os.Create("sitemap.xml")
	if err != nil {
		fmt.Printf("Error creating sitemap.xml: %v", err)
	}
	defer file.Close()

	s := sitemap.New()
	sitemap.LastMod = time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	sitemap.ChangeFreq = "monthly"

	url := "https://example.com"
	err = s.Generate(file, &url)
	if err != nil {
		fmt.Printf("Error generating sitemap: %v", err)
	}
}
```

sitemap.xml

```xml
<?xml version="1.0" encoding="UTF-8"?>
<urlset xmlns="http://www.sitemaps.org/schemas/sitemap/0.9">
  <url>
    <loc>https://example.com/</loc>
    <lastmod>2023-01-01T00:00:00Z</lastmod>
    <priority>1.0</priority>
	<changefreq>monthly</changefreq>
  </url>
</urlset>
```
