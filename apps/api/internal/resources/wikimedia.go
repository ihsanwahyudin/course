// Package resources finds real-world material for a lesson: freely licensed
// pictures from Wikimedia Commons. (Web links come from the AI provider's
// search tool; curated links come from the course files.)
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Image is a Commons picture with what's needed to credit it.
type Image struct {
	Title   string `json:"title"`
	Thumb   string `json:"thumb"`   // upload/thumb.wikimedia.org URL, ~480px wide
	PageURL string `json:"pageUrl"` // the Commons file page (licence details)
	Author  string `json:"author"`
	License string `json:"license"`
	Width   int    `json:"width"`
	Height  int    `json:"height"`
}

// Commons searches Wikimedia Commons. Requests only ever go to BaseURL, so a
// query can't make the server fetch arbitrary URLs.
type Commons struct {
	BaseURL   string // default https://commons.wikimedia.org/w/api.php
	UserAgent string
	HTTP      *http.Client
}

func NewCommons(contact string) *Commons {
	return &Commons{
		BaseURL:   "https://commons.wikimedia.org/w/api.php",
		UserAgent: "FluentPath/1.0 (English learning app; " + contact + ")",
		HTTP:      &http.Client{Timeout: 8 * time.Second},
	}
}

var tagRe = regexp.MustCompile(`<[^>]*>`)

func plain(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(tagRe.ReplaceAllString(s, " "))), " ")
}

// allowedImageHosts are where Commons serves files from.
var allowedImageHosts = map[string]bool{"upload.wikimedia.org": true, "thumb.wikimedia.org": true}

func cleanImageURL(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || !allowedImageHosts[u.Hostname()] {
		return ""
	}
	u.RawQuery = "" // drop utm_ tracking
	return u.String()
}

// Search returns up to limit photos or drawings matching query.
func (c *Commons) Search(ctx context.Context, query string, limit int) ([]Image, error) {
	q := url.Values{
		"action": {"query"}, "format": {"json"}, "formatversion": {"2"},
		"generator": {"search"}, "gsrnamespace": {"6"}, "gsrsearch": {query + " filetype:bitmap"},
		"gsrlimit": {fmt.Sprint(limit * 2)}, "prop": {"imageinfo"},
		"iiprop": {"url|extmetadata|mime"}, "iiurlwidth": {"480"},
		"iiextmetadatafilter": {"LicenseShortName|Artist|ObjectName"},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"?"+q.Encode(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.UserAgent)
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("commons status %d", resp.StatusCode)
	}
	var body struct {
		Query struct {
			Pages []struct {
				Index     int    `json:"index"`
				Title     string `json:"title"`
				ImageInfo []struct {
					ThumbURL       string `json:"thumburl"`
					ThumbWidth     int    `json:"thumbwidth"`
					ThumbHeight    int    `json:"thumbheight"`
					DescriptionURL string `json:"descriptionurl"`
					Mime           string `json:"mime"`
					ExtMetadata    map[string]struct {
						Value string `json:"value"`
					} `json:"extmetadata"`
				} `json:"imageinfo"`
			} `json:"pages"`
		} `json:"query"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&body); err != nil {
		return nil, fmt.Errorf("decode commons response: %w", err)
	}
	pages := body.Query.Pages
	sort.Slice(pages, func(i, j int) bool { return pages[i].Index < pages[j].Index })
	var out []Image
	for _, p := range pages {
		if len(p.ImageInfo) == 0 || len(out) >= limit {
			continue
		}
		ii := p.ImageInfo[0]
		if ii.Mime != "image/jpeg" && ii.Mime != "image/png" {
			continue
		}
		thumb := cleanImageURL(ii.ThumbURL)
		page, err := url.Parse(ii.DescriptionURL)
		license := plain(ii.ExtMetadata["LicenseShortName"].Value)
		if thumb == "" || err != nil || page.Scheme != "https" || page.Hostname() != "commons.wikimedia.org" || license == "" {
			continue
		}
		title := plain(ii.ExtMetadata["ObjectName"].Value)
		if title == "" {
			title = strings.TrimSuffix(strings.TrimPrefix(p.Title, "File:"), ".jpg")
		}
		author := plain(ii.ExtMetadata["Artist"].Value)
		if author == "" {
			author = "Wikimedia Commons"
		}
		out = append(out, Image{Title: title, Thumb: thumb, PageURL: page.String(), Author: author, License: license,
			Width: ii.ThumbWidth, Height: ii.ThumbHeight})
	}
	return out, nil
}
