package main

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"errors"
	"regexp"
	"strings"
	"time"

	"howett.net/plist"
)

var (
	maxIPABytes            int64 = 2 << 30
	maxMetadataBytes       int64 = 4 << 20
	maxZipEntries                = 20000
	maxApplicationsPerChat       = 50
	maxConcurrentIPA             = 2
)

var ipaSlots = make(chan struct{}, maxConcurrentIPA)

var (
	payloadInfoPlist = regexp.MustCompile(`^Payload/[^/]+\.app/Info\.plist$`)
	payloadProvision = regexp.MustCompile(`^Payload/[^/]+\.app/embedded\.mobileprovision$`)
	provisionPlist   = regexp.MustCompile(`<plist[\s\S]*?</plist>`)
	provisionCreated = regexp.MustCompile(`<key>CreationDate</key>[\s\S]*?<date>([\s\S]*?)</date>`)
	provisionExpires = regexp.MustCompile(`<key>ExpirationDate</key>[\s\S]*?<date>([\s\S]*?)</date>`)
	provisionTeam    = regexp.MustCompile(`<key>TeamName</key>[\s\S]*?<string>([\s\S]*?)</string>`)
)

func acquireIPASlot() bool {
	select {
	case ipaSlots <- struct{}{}:
		return true
	default:
		return false
	}
}

func releaseIPASlot() {
	<-ipaSlots
}

func isIPAFileName(name string) bool {
	if name == "" || strings.ContainsAny(name, "/\\\x00") {
		return false
	}
	return strings.HasSuffix(strings.ToLower(name), ".ipa")
}

func zipEntryKind(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	name = strings.TrimPrefix(name, "/")
	for strings.HasPrefix(name, "./") {
		name = strings.TrimPrefix(name, "./")
	}
	if strings.Contains(name, "..") {
		return ""
	}
	switch {
	case payloadInfoPlist.MatchString(name):
		return "info"
	case payloadProvision.MatchString(name):
		return "prov"
	default:
		return ""
	}
}

func parseIPA(path string) (Application, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return Application{}, userErr("The file is not a valid IPA.")
	}
	defer zr.Close()
	if len(zr.File) > maxZipEntries {
		return Application{}, userErr("The IPA contains too many files.")
	}
	var app Application
	var gotInfo, gotProv bool
	for _, file := range zr.File {
		if file.FileInfo().IsDir() {
			continue
		}
		kind := zipEntryKind(file.Name)
		if kind == "" {
			continue
		}
		if (kind == "info" && gotInfo) || (kind == "prov" && gotProv) {
			continue
		}
		if file.UncompressedSize64 > uint64(maxMetadataBytes) {
			return Application{}, userErr("The IPA metadata is too large.")
		}
		rc, err := file.Open()
		if err != nil {
			return Application{}, userErr("The file is not a valid IPA.")
		}
		data, readErr := readLimited(rc, maxMetadataBytes)
		rc.Close()
		if readErr != nil {
			if errors.Is(readErr, errTooLarge) {
				return Application{}, userErr("The IPA metadata is too large.")
			}
			return Application{}, userErr("The file is not a valid IPA.")
		}
		switch kind {
		case "info":
			if err := applyInfoPlist(&app, data); err != nil {
				return Application{}, err
			}
			gotInfo = true
		case "prov":
			applyProvision(&app, data)
			gotProv = true
		}
		if gotInfo && gotProv {
			break
		}
	}
	if !gotInfo || app.Package == "" {
		return Application{}, userErr("The IPA is missing a valid Info.plist.")
	}
	return app, nil
}

func applyInfoPlist(app *Application, data []byte) (err error) {
	defer func() {
		if rec := recover(); rec != nil {
			err = userErr("The IPA Info.plist is invalid.")
		}
	}()
	if xmlNestingTooDeep(data, 32) {
		return userErr("The IPA Info.plist is invalid.")
	}
	plistV := map[string]any{}
	if _, err = plist.Unmarshal(data, &plistV); err != nil {
		return userErr("The IPA Info.plist is invalid.")
	}
	displayName := sanitizeText(plistString(plistV, "CFBundleDisplayName"))
	if displayName == "" {
		displayName = sanitizeText(plistString(plistV, "CFBundleExecutable"))
	}
	app.Package = sanitizeText(plistString(plistV, "CFBundleIdentifier"))
	if app.Package == "" {
		return userErr("The IPA is missing a bundle identifier.")
	}
	app.Name = app.Package
	if displayName != "" {
		app.Name = displayName
	}
	app.Version = sanitizeText(plistString(plistV, "CFBundleShortVersionString"))
	return nil
}

func plistString(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

func xmlNestingTooDeep(data []byte, maxDepth int) bool {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '<' {
		return false
	}
	dec := xml.NewDecoder(bytes.NewReader(trimmed))
	depth := 0
	for {
		tok, err := dec.Token()
		if err != nil {
			return false
		}
		switch tok.(type) {
		case xml.StartElement:
			depth++
			if depth > maxDepth {
				return true
			}
		case xml.EndElement:
			if depth > 0 {
				depth--
			}
		}
	}
}

func applyProvision(app *Application, data []byte) {
	group := provisionPlist.FindSubmatch(data)
	if len(group) == 0 {
		return
	}
	mobileprovision := string(group[0])
	if g := provisionCreated.FindStringSubmatch(mobileprovision); len(g) > 1 {
		app.CertCreatedAt, _ = time.Parse(time.RFC3339, strings.TrimSpace(g[1]))
	}
	if g := provisionExpires.FindStringSubmatch(mobileprovision); len(g) > 1 {
		app.CertExpiredAt, _ = time.Parse(time.RFC3339, strings.TrimSpace(g[1]))
	}
	if g := provisionTeam.FindStringSubmatch(mobileprovision); len(g) > 1 {
		app.CertOrg = sanitizeText(g[1])
	}
}
