package main

import (
	"crypto/x509"
	"encoding/pem"
	"strings"
	"unicode"

	"go.uber.org/zap"
)

func sanitizeUser(s string) string {
	return strings.Map(
		func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		},
		s,
	)
}

func isAdmin(user string) bool {
	for _, admin := range configuration.Admins {
		if user == admin {
			return true
		}
	}
	return false
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func getUserFromCert(cert string) string {
	block, _ := pem.Decode([]byte(cert))
	if block == nil {
		log.Warn("failed to parse PEM block containing the public key")
		return ""
	}
	pub, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		log.Error("failed to parse certificate", zap.Error(err))
		return ""
	}
	return pub.Subject.CommonName
}

func isAuthorized(user string, uri string, host string) bool {
	if isAdmin(user) {
		return true
	}

	if locations, ok := configuration.Hosts[host]; ok {
		for location, users := range locations.Location {
			if strings.HasPrefix(uri, location) {
				log.Debug("URI matches LocationRule",
					zap.String("uri", uri),
					zap.String("locationRule", location))
				if stringInSlice(user, users.Users) {
					log.Debug("Authenticating user",
						zap.String("user", user),
						zap.Strings("allowedUsers", users.Users))
					return true
				}
			}
		}
	}

	log.Info("user not Authorized",
		zap.String("user", user))
	return false
}
