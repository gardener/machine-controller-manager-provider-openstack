// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package client

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// List of headers that contain sensitive data.
var defaultSensitiveHeaders = map[string]struct{}{
	"x-auth-token":                    {},
	"x-auth-key":                      {},
	"x-service-token":                 {},
	"x-storage-token":                 {},
	"x-account-meta-temp-url-key":     {},
	"x-account-meta-temp-url-key-2":   {},
	"x-container-meta-temp-url-key":   {},
	"x-container-meta-temp-url-key-2": {},
	"set-cookie":                      {},
	"x-subject-token":                 {},
	"authorization":                   {},
}

type loggerInterface interface {
	Printf(format string, args ...interface{})
}

// noopLogger is a default noop logger satisfies the Logger interface
type noopLogger struct{}

// Printf is a default noop method
func (noopLogger) Printf(_ string, _ ...interface{}) {}

type loggingRoundTripper struct {
	Rt     http.RoundTripper
	Logger loggerInterface
}

// RoundTrip is the implementation of the http.RoundTripper interface.
func (rt *loggingRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	defer func() {
		if req.Body != nil {
			_ = req.Body.Close()
		}
	}()

	var err error

	if rt.Logger != nil {
		rt.log().Printf("OpenStack Request URL: %s %s", req.Method, req.URL)
		rt.log().Printf("OpenStack Request Headers:\n%s", formatHeaders(req.Header, "\n"))

		// could log request body (JSON) here
	}

	// this is concurrency safe
	ort := rt.Rt
	if ort == nil {
		return nil, fmt.Errorf("loggingRoundTrippers RoundTripper is nil, aborting")
	}
	response, err := ort.RoundTrip(req)

	// could implement retries here

	if rt.Logger != nil {
		rt.log().Printf("OpenStack Response Code: %d", response.StatusCode)
		rt.log().Printf("OpenStack Response Headers:\n%s", formatHeaders(response.Header, "\n"))

		// could log response (JSON) here
	}

	return response, err
}

func (rt *loggingRoundTripper) log() loggerInterface {
	// this is concurrency safe
	l := rt.Logger
	if l == nil {
		// noop is used, when logger pointer has been set to nil
		return &noopLogger{}
	}
	return l
}

// formatHeaders converts standard http.Header type to a string with separated headers.
// It will hide data of sensitive headers.
func formatHeaders(headers http.Header, separator string) string {
	redactedHeaders := hideSensitiveHeadersData(headers)
	sort.Strings(redactedHeaders)

	return strings.Join(redactedHeaders, separator)
}

func hideSensitiveHeadersData(headers http.Header) []string {
	result := make([]string, len(headers))
	headerIdx := 0

	for header, data := range headers {
		v := strings.ToLower(header)
		if _, ok := defaultSensitiveHeaders[v]; ok {
			result[headerIdx] = fmt.Sprintf("%s: %s", header, "***")
		} else {
			result[headerIdx] = fmt.Sprintf("%s: %s", header, strings.Join(data, " "))
		}
		headerIdx++
	}

	return result
}
