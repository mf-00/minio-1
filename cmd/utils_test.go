/*
 * Minio Cloud Storage, (C) 2016 Minio, Inc.
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
	"fmt"
	"net/http"
	"reflect"
	"runtime"
	"testing"
)

// Tests http.Header clone.
func TestCloneHeader(t *testing.T) {
	headers := []http.Header{
		{
			"Content-Type":   {"text/html; charset=UTF-8"},
			"Content-Length": {"0"},
		},
		{
			"Content-Length": {"0", "1", "2"},
		},
		{
			"Expires":          {"-1"},
			"Content-Length":   {"0"},
			"Content-Encoding": {"gzip"},
		},
	}
	for i, header := range headers {
		clonedHeader := cloneHeader(header)
		if !reflect.DeepEqual(header, clonedHeader) {
			t.Errorf("Test %d failed", i+1)
		}
	}
}

// Tests check duplicates function.
func TestCheckDuplicates(t *testing.T) {
	tests := []struct {
		list       []string
		err        error
		shouldPass bool
	}{
		// Test 1 - for '/tmp/1' repeated twice.
		{
			list:       []string{"/tmp/1", "/tmp/1", "/tmp/2", "/tmp/3"},
			err:        fmt.Errorf("Duplicate key: \"/tmp/1\" found of count: \"2\""),
			shouldPass: false,
		},
		// Test 2 - for '/tmp/1' repeated thrice.
		{
			list:       []string{"/tmp/1", "/tmp/1", "/tmp/1", "/tmp/3"},
			err:        fmt.Errorf("Duplicate key: \"/tmp/1\" found of count: \"3\""),
			shouldPass: false,
		},
		// Test 3 - empty string.
		{
			list:       []string{""},
			err:        errInvalidArgument,
			shouldPass: false,
		},
		// Test 4 - empty string.
		{
			list:       nil,
			err:        errInvalidArgument,
			shouldPass: false,
		},
		// Test 5 - non repeated strings.
		{
			list:       []string{"/tmp/1", "/tmp/2", "/tmp/3"},
			err:        nil,
			shouldPass: true,
		},
	}

	// Validate if function runs as expected.
	for i, test := range tests {
		err := checkDuplicates(test.list)
		if test.shouldPass && err != test.err {
			t.Errorf("Test: %d, Expected %s got %s", i+1, test.err, err)
		}
		if !test.shouldPass && err.Error() != test.err.Error() {
			t.Errorf("Test: %d, Expected %s got %s", i+1, test.err, err)
		}
	}
}

// Tests maximum object size.
func TestMaxObjectSize(t *testing.T) {
	sizes := []struct {
		isMax bool
		size  int64
	}{
		// Test - 1 - maximum object size.
		{
			true,
			maxObjectSize + 1,
		},
		// Test - 2 - not maximum object size.
		{
			false,
			maxObjectSize - 1,
		},
	}
	for i, s := range sizes {
		isMax := isMaxObjectSize(s.size)
		if isMax != s.isMax {
			t.Errorf("Test %d: Expected %t, got %t", i+1, s.isMax, isMax)
		}
	}
}

// Test urlPathSplit.
func TestURLPathSplit(t *testing.T) {
	type test struct {
		urlPath    string
		bucketName string
		prefixName string
	}

	testCases := []test{
		{
			urlPath:    "/b/c/",
			bucketName: "b",
			prefixName: "c/",
		},
		{
			urlPath:    "c/aa",
			bucketName: "c",
			prefixName: "aa",
		},
		{
			urlPath:    "",
			bucketName: "",
			prefixName: "",
		},
		{
			urlPath:    "/b",
			bucketName: "b",
			prefixName: "",
		},
	}

	for i, testCase := range testCases {
		bucketName, prefixName := urlPathSplit(testCase.urlPath)
		if bucketName != testCase.bucketName {
			t.Errorf("Tets %d: Expected %s, %s", i+1, testCase.bucketName, bucketName)
		}
		if prefixName != testCase.prefixName {
			t.Errorf("Tets %d: Expected %s, %s", i+1, testCase.bucketName, bucketName)
		}
	}
}

// Tests minimum allowed part size.
func TestMinAllowedPartSize(t *testing.T) {
	sizes := []struct {
		isMin bool
		size  int64
	}{
		// Test - 1 - within minimum part size.
		{
			true,
			minPartSize + 1,
		},
		// Test - 2 - smaller than minimum part size.
		{
			false,
			minPartSize - 1,
		},
	}

	for i, s := range sizes {
		isMin := isMinAllowedPartSize(s.size)
		if isMin != s.isMin {
			t.Errorf("Test %d: Expected %t, got %t", i+1, s.isMin, isMin)
		}
	}
}

// Tests maximum allowed part number.
func TestMaxPartID(t *testing.T) {
	sizes := []struct {
		isMax bool
		partN int
	}{
		// Test - 1 part number within max part number.
		{
			false,
			maxPartID - 1,
		},
		// Test - 2 part number bigger than max part number.
		{
			true,
			maxPartID + 1,
		},
	}

	for i, s := range sizes {
		isMax := isMaxPartID(s.partN)
		if isMax != s.isMax {
			t.Errorf("Test %d: Expected %t, got %t", i+1, s.isMax, isMax)
		}
	}
}

// Tests fetch local address.
func TestLocalAddress(t *testing.T) {
	if runtime.GOOS == "windows" {
		return
	}
	// need to set this to avoid stale values from other tests.
	globalMinioPort = 9000
	testCases := []struct {
		srvCmdConfig serverCmdConfig
		localAddr    string
	}{
		// Test 1 - local address is found.
		{
			srvCmdConfig: serverCmdConfig{
				isDistXL: true,
				disks: []string{
					"localhost:/mnt/disk1",
					"1.1.1.2:/mnt/disk2",
					"1.1.2.1:/mnt/disk3",
					"1.1.2.2:/mnt/disk4",
				},
			},
			localAddr: fmt.Sprintf("localhost:%d", globalMinioPort),
		},
		// Test 2 - local address is everything.
		{
			srvCmdConfig: serverCmdConfig{
				isDistXL: false,
				disks: []string{
					"/mnt/disk1",
					"/mnt/disk2",
					"/mnt/disk3",
					"/mnt/disk4",
				},
			},
			localAddr: fmt.Sprintf(":%d", globalMinioPort),
		},
		// Test 3 - local address is not found.
		{
			srvCmdConfig: serverCmdConfig{
				isDistXL: true,
				disks: []string{
					"1.1.1.1:/mnt/disk1",
					"1.1.1.2:/mnt/disk2",
					"1.1.2.1:/mnt/disk3",
					"1.1.2.2:/mnt/disk4",
				},
			},
			localAddr: "",
		},
	}

	// Validates fetching local address.
	for i, testCase := range testCases {
		localAddr := getLocalAddress(testCase.srvCmdConfig)
		if localAddr != testCase.localAddr {
			t.Fatalf("Test %d: Expected %s, got %s", i+1, testCase.localAddr, localAddr)
		}
	}

}
