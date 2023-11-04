package main

import (
	"errors"
	"github.com/life4/genesis/lambdas"
	"github.com/life4/genesis/slices"
	"strconv"
	"time"
)

func BuildAppInfoTemplate(application Application) string {
	return `
<b>App Name:</b> ` + application.Name + `
<b>Package Name:</b> ` + application.Package + `
<b>Version:</b> ` + application.Version + `
<b>Certificate Team Name:</b> ` + application.CertOrg + `
<b>Certificate Creation Date:</b> ` + application.CertCreatedAt.Format(time.DateTime) + `
<b>Certificate Expiration Date:</b> ` + application.CertExpiredAt.Format(time.DateTime) + `
<a href="` + application.IPA + `">Download IPA</a>
<a href="` + application.InstallPage + `">Install</a>
`
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
	for _, app := range outApps {
		res += `<a href="https://t.me/` + botUsername + `?start=app_` + app.UUID + `">` + app.Name +
			`</a>, created at ` + app.CreatedAt.Format(time.DateTime) +
			`, certificate expires at ` + app.CertExpiredAt.Format(time.DateTime) +
			` [<a href="https://t.me/` + botUsername + `?start=del_` + app.UUID + `">Delete</a>]` +
			"\n"
	}
	return res, nil
}
