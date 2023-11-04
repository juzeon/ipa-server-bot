package main

import (
	"github.com/life4/genesis/slices"
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
func BuildAppListTemplate(apps []Application, botUsername string) string {
	apps = slices.Reverse(slices.SortBy(apps, func(el Application) int64 {
		return el.CreatedAt.Unix()
	}))
	res := "<b>List of your applications:</b>\n"
	for _, app := range apps {
		res += `<a href="https://t.me/` + botUsername + `?start=app_` + app.UUID + `">` + app.Name +
			`</a>, created at ` + app.CreatedAt.Format(time.DateTime) +
			`, certificate expires at ` + app.CertExpiredAt.Format(time.DateTime) +
			` [<a href="https://t.me/` + botUsername + `?start=del_` + app.UUID + `">Delete</a>]` +
			"\n"
	}
	return res
}
