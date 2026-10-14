package resources

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCommonsSearch(t *testing.T) {
	var gotUA, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotQuery = r.UserAgent(), r.URL.Query().Get("gsrsearch")
		_, _ = io.WriteString(w, `{"query":{"pages":[
		 {"index":2,"title":"File:B.png","imageinfo":[{"thumburl":"https://upload.wikimedia.org/b.png?utm_source=x","thumbwidth":480,"thumbheight":300,
		  "descriptionurl":"https://commons.wikimedia.org/wiki/File:B.png","mime":"image/png",
		  "extmetadata":{"LicenseShortName":{"value":"CC BY-SA 4.0"},"Artist":{"value":"<a href=\"//x\">Ann &amp; Bo</a>"},"ObjectName":{"value":"Blue bus"}}}]},
		 {"index":1,"title":"File:A.jpg","imageinfo":[{"thumburl":"https://thumb.wikimedia.org/a.jpg","descriptionurl":"https://commons.wikimedia.org/wiki/File:A.jpg","mime":"image/jpeg",
		  "extmetadata":{"LicenseShortName":{"value":"CC0"}}}]},
		 {"index":3,"title":"File:Evil.jpg","imageinfo":[{"thumburl":"https://evil.example/a.jpg","descriptionurl":"https://commons.wikimedia.org/wiki/File:Evil.jpg","mime":"image/jpeg",
		  "extmetadata":{"LicenseShortName":{"value":"CC0"}}}]},
		 {"index":4,"title":"File:NoLicence.jpg","imageinfo":[{"thumburl":"https://upload.wikimedia.org/n.jpg","descriptionurl":"https://commons.wikimedia.org/wiki/File:N.jpg","mime":"image/jpeg","extmetadata":{}}]},
		 {"index":5,"title":"File:Clip.webm","imageinfo":[{"thumburl":"https://upload.wikimedia.org/c.jpg","descriptionurl":"https://commons.wikimedia.org/wiki/File:C.webm","mime":"video/webm",
		  "extmetadata":{"LicenseShortName":{"value":"CC0"}}}]}
		]}}`)
	}))
	defer srv.Close()
	c := NewCommons("test")
	c.BaseURL = srv.URL
	imgs, err := c.Search(context.Background(), "blue bus", 4)
	if err != nil {
		t.Fatal(err)
	}
	if len(imgs) != 2 || imgs[0].Title != "A" || imgs[0].Author != "Wikimedia Commons" || imgs[1].Thumb != "https://upload.wikimedia.org/b.png" ||
		imgs[1].Author != "Ann & Bo" || imgs[1].License != "CC BY-SA 4.0" {
		t.Fatalf("images %+v", imgs)
	}
	if !strings.HasPrefix(gotUA, "FluentPath/1.0") || gotQuery != "blue bus filetype:bitmap" {
		t.Fatalf("request: ua=%q q=%q", gotUA, gotQuery)
	}
}
