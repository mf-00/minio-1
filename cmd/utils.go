/*
 * Minio Cloud Storage, (C) 2015 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"encoding/json"

	"github.com/pkg/profile"
)

// make a copy of http.Header
func cloneHeader(h http.Header) http.Header {
	h2 := make(http.Header, len(h))
	for k, vv := range h {
		vv2 := make([]string, len(vv))
		copy(vv2, vv)
		h2[k] = vv2

	}
	return h2
}

// checkDuplicates - function to validate if there are duplicates in a slice of strings.
func checkDuplicates(list []string) error {
	// Empty lists are not allowed.
	if len(list) == 0 {
		return errInvalidArgument
	}
	// Empty keys are not allowed.
	for _, key := range list {
		if key == "" {
			return errInvalidArgument
		}
	}
	listMaps := make(map[string]int)
	// Navigate through each configs and count the entries.
	for _, key := range list {
		listMaps[key]++
	}
	// Validate if there are any duplicate counts.
	for key, count := range listMaps {
		if count != 1 {
			return fmt.Errorf("Duplicate key: \"%s\" found of count: \"%d\"", key, count)
		}
	}
	// No duplicates.
	return nil
}

// splits network path into its components Address and Path.
func splitNetPath(networkPath string) (netAddr, netPath string, err error) {
	if runtime.GOOS == "windows" {
		if volumeName := filepath.VolumeName(networkPath); volumeName != "" {
			return "", networkPath, nil
		}
	}
	networkParts := strings.SplitN(networkPath, ":", 2)
	switch {
	case len(networkParts) == 1:
		return "", networkPath, nil
	case networkParts[1] == "":
		return "", "", &net.AddrError{Err: "Missing path in network path", Addr: networkPath}
	case networkParts[0] == "":
		return "", "", &net.AddrError{Err: "Missing address in network path", Addr: networkPath}
	case !filepath.IsAbs(networkParts[1]):
		return "", "", &net.AddrError{Err: "Network path should be absolute", Addr: networkPath}
	}
	return networkParts[0], networkParts[1], nil
}

// Find local node through the command line arguments. Returns in
// `host:port` format.
func getLocalAddress(srvCmdConfig serverCmdConfig) string {
	if !srvCmdConfig.isDistXL {
		return fmt.Sprintf(":%d", globalMinioPort)
	}
	for _, export := range srvCmdConfig.disks {
		// Validates if remote disk is local.
		if isLocalStorage(export) {
			var host string
			if idx := strings.LastIndex(export, ":"); idx != -1 {
				host = export[:idx]
			}
			return fmt.Sprintf("%s:%d", host, globalMinioPort)
		}
	}
	return ""
}

// xmlDecoder provide decoded value in xml.
func xmlDecoder(body io.Reader, v interface{}, size int64) error {
	var lbody io.Reader
	if size > 0 {
		lbody = io.LimitReader(body, size)
	} else {
		lbody = body
	}
	d := xml.NewDecoder(lbody)
	return d.Decode(v)
}

// checkValidMD5 - verify if valid md5, returns md5 in bytes.
func checkValidMD5(md5 string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(strings.TrimSpace(md5))
}

/// http://docs.aws.amazon.com/AmazonS3/latest/dev/UploadingObjects.html
const (
	// maximum object size per PUT request is 5GiB
	maxObjectSize = 1024 * 1024 * 1024 * 5
	// minimum Part size for multipart upload is 5MB
	minPartSize = 1024 * 1024 * 5
	// maximum Part ID for multipart upload is 10000 (Acceptable values range from 1 to 10000 inclusive)
	maxPartID = 10000
)

// isMaxObjectSize - verify if max object size
func isMaxObjectSize(size int64) bool {
	return size > maxObjectSize
}

// Check if part size is more than or equal to minimum allowed size.
func isMinAllowedPartSize(size int64) bool {
	return size >= minPartSize
}

// isMaxPartNumber - Check if part ID is greater than the maximum allowed ID.
func isMaxPartID(partID int) bool {
	return partID > maxPartID
}

func contains(stringList []string, element string) bool {
	for _, e := range stringList {
		if e == element {
			return true
		}
	}
	return false
}

// urlPathSplit - split url path into bucket and object components.
func urlPathSplit(urlPath string) (bucketName, prefixName string) {
	if urlPath == "" {
		return urlPath, ""
	}
	urlPath = strings.TrimPrefix(urlPath, "/")
	i := strings.Index(urlPath, "/")
	if i != -1 {
		return urlPath[:i], urlPath[i+1:]
	}
	return urlPath, ""
}

// Starts a profiler returns nil if profiler is not enabled, caller needs to handle this.
func startProfiler(profiler string) interface {
	Stop()
} {
	// Set ``MINIO_PROFILE_DIR`` to the directory where profiling information should be persisted
	profileDir := os.Getenv("MINIO_PROFILE_DIR")
	// Enable profiler if ``MINIO_PROFILER`` is set. Supported options are [cpu, mem, block].
	switch profiler {
	case "cpu":
		return profile.Start(profile.CPUProfile, profile.NoShutdownHook, profile.ProfilePath(profileDir))
	case "mem":
		return profile.Start(profile.MemProfile, profile.NoShutdownHook, profile.ProfilePath(profileDir))
	case "block":
		return profile.Start(profile.BlockProfile, profile.NoShutdownHook, profile.ProfilePath(profileDir))
	default:
		return nil
	}
}

// Global profiler to be used by service go-routine.
var globalProfiler interface {
	Stop()
}

// dump the request into a string in JSON format.
func dumpRequest(r *http.Request) string {
	header := cloneHeader(r.Header)
	header.Set("Host", r.Host)
	req := struct {
		Method string      `json:"method"`
		Path   string      `json:"path"`
		Query  string      `json:"query"`
		Header http.Header `json:"header"`
	}{r.Method, r.URL.Path, r.URL.RawQuery, header}
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return "<error dumping request>"
	}
	return string(jsonBytes)
}
