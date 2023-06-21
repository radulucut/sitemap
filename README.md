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
	// s.IgnoreQuery = true
	// s.IgnoreFragment = true
	// s.Verbose = false

	url := "https://example.com"
	err = s.Generate(file, &url)
	if err != nil {
		fmt.Printf("Error generating sitemap: %v", err)
	}
}
```
