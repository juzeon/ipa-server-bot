package main

import "time"

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
