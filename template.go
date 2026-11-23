package main

import (
	"errors"
	"html"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/life4/genesis/lambdas"
	"github.com/life4/genesis/slices"
)

func sanitizeText(s string) string {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		if r < 0x20 || r == 0x7f {
			return -1
		}
		switch r {
		case 0x202A, 0x202B, 0x202C, 0x202D, 0x202E,
			0x2066, 0x2067, 0x2068, 0x2069,
			0x200E, 0x200F, 0xFEFF:
			return -1
		}
		return r
	}, s)
	s = strings.Join(strings.Fields(s), " ")
	const maxRunes = 256
	rs := []rune(s)
	if len(rs) > maxRunes {
		s = string(rs[:maxRunes])
	}
	return s
}

func htmlText(s string) string {
	return html.EscapeString(sanitizeText(s))
}

func sanitizeHTTPURL(raw string) (string, error) {
	if strings.ContainsAny(raw, "\r\n\t\\") {
		return "", errors.New("invalid url")
	}
	u, err := url.Parse(raw)
	if err != nil || u == nil {
		return "", errors.New("invalid url")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("invalid url")
	}
	if u.Hostname() == "" || u.User != nil {
		return "", errors.New("invalid url")
	}
	return u.String(), nil
}

func htmlHref(raw string) (string, bool) {
	u, err := sanitizeHTTPURL(raw)
	if err != nil {
		return "", false
	}
	return html.EscapeString(u), true
}

func BuildAppInfoTemplate(application Application) string {
	var b strings.Builder
	b.WriteString("\n<b>App Name:</b> " + htmlText(application.Name) + "\n")
	b.WriteString("<b>Package Name:</b> " + htmlText(application.Package) + "\n")
	b.WriteString("<b>Version:</b> " + htmlText(application.Version) + "\n")
	b.WriteString("<b>Certificate Team Name:</b> " + htmlText(application.CertOrg) + "\n")
	b.WriteString("<b>Certificate Creation Date:</b> " + application.CertCreatedAt.Format(time.DateTime) + "\n")
	b.WriteString("<b>Certificate Expiration Date:</b> " + application.CertExpiredAt.Format(time.DateTime) + "\n")
	if href, ok := htmlHref(application.IPA); ok {
		b.WriteString(`<a href="` + href + `">Download IPA</a>` + "\n")
	}
	if href, ok := htmlHref(application.InstallPage); ok {
		b.WriteString(`<a href="` + href + `">Install</a>` + "\n")
	}
	return b.String()
}
func BuildAppListTemplate(apps []Application, botUsername string, page int) (string, error) {
	const perPage = 10
	apps = slices.Reverse(slices.SortBy(apps, func(el Application) int64 {
		return el.CreatedAt.Unix()
	}))
	totalPage := len(apps) / perPage
	if len(apps)%perPage != 0 {
		totalPage++
	}
	outApps := apps
	if totalPage == 0 {
		return "", errors.New("you have no applications currently")
	}
	if page < 1 {
		return "", errors.New("this is the first page")
	}
	if page > totalPage {
		return "", errors.New("this is the last page")
	}
	if len(apps) > perPage {
		outApps = apps[(page-1)*perPage : lambdas.Min(perPage*page, len(apps))]
	}
	res := "<b>List of your applications:</b>\n" +
		"(Page: " + strconv.Itoa(page) + "/" + strconv.Itoa(totalPage) + ")\n"
	user := html.EscapeString(url.PathEscape(botUsername))
	for _, app := range outApps {
		id := html.EscapeString(url.PathEscape(app.UUID))
		res += `<a href="https://t.me/` + user + `?start=app_` + id + `">` + htmlText(app.Name) +
			`</a>, created at ` + app.CreatedAt.Format(time.DateTime) +
			`, certificate expires at ` + app.CertExpiredAt.Format(time.DateTime) +
			` [<a href="https://t.me/` + user + `?start=del_` + id + `">Delete</a>]` +
			"\n"
	}
	return res, nil
}
