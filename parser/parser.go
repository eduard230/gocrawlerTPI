package parser

import (
	"io"
	"strings"

	"golang.org/x/net/html"
)

// PageInfo contains extracted information from a page
type PageInfo struct {
	Title       string
	Description string
	Links       []string
}

// Parse extracts information from HTML content
func Parse(body io.Reader, baseURL string) (*PageInfo, error) {
	doc, err := html.Parse(body)
	if err != nil {
		return nil, err
	}

	info := &PageInfo{
		Links: make([]string, 0),
	}

	// Traverse DOM and extract data
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "title":
				if n.FirstChild != nil {
					info.Title = strings.TrimSpace(n.FirstChild.Data)
				}
			case "meta":
				// Extract meta description
				var name, content string
				for _, attr := range n.Attr {
					if attr.Key == "name" && attr.Val == "description" {
						name = attr.Val
					}
					if attr.Key == "content" {
						content = attr.Val
					}
				}
				if name == "description" {
					info.Description = content
				}
			case "a":
				// Extract links
				for _, attr := range n.Attr {
					if attr.Key == "href" {
						href := strings.TrimSpace(attr.Val)
						if href != "" && !strings.HasPrefix(href, "#") && !strings.HasPrefix(href, "javascript:") {
							info.Links = append(info.Links, href)
						}
					}
				}
			}
		}

		// Recursively traverse children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}

	traverse(doc)

	// Remove duplicate links
	info.Links = uniqueStrings(info.Links)

	return info, nil
}

// uniqueStrings removes duplicates from string slice
func uniqueStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, item := range slice {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}
