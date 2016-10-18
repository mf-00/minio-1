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
	"sync"

	humanize "github.com/dustin/go-humanize"
	"github.com/minio/mc/pkg/console"
)

// Helper to generate integer sequences into a friendlier user consumable format.
func int2Str(i int, t int) string {
	if i < 10 {
		if t < 10 {
			return fmt.Sprintf("0%d/0%d", i, t)
		}
		return fmt.Sprintf("0%d/%d", i, t)
	}
	return fmt.Sprintf("%d/%d", i, t)
}

// Print a given message once.
type printOnceFunc func(msg string)

// Print once is a constructor returning a function printing once.
// internally print uses sync.Once to perform exactly one action.
func printOnceFn() printOnceFunc {
	var once sync.Once
	return func(msg string) {
		once.Do(func() { console.Println(msg) })
	}
}

// Prints custom message when healing is required for XL and Distributed XL backend.
func printHealMsg(firstEndpoint string, storageDisks []StorageAPI, fn printOnceFunc) {
	msg := getHealMsg(firstEndpoint, storageDisks)
	fn(msg)
}

// Constructs a formatted heal message, when cluster is found to be in state where it requires healing.
// healing is optional, server continues to initialize object layer after printing this message.
// it is upto the end user to perform a heal if needed.
func getHealMsg(firstEndpoint string, storageDisks []StorageAPI) string {
	msg := fmt.Sprintln("\nData volume requires HEALING. Please run the following command:")
	msg += "MINIO_ACCESS_KEY=%s "
	msg += "MINIO_SECRET_KEY=%s "
	msg += "minio control heal %s"
	creds := serverConfig.GetCredential()
	msg = fmt.Sprintf(msg, creds.AccessKeyID, creds.SecretAccessKey, firstEndpoint)
	disksInfo, _, _ := getDisksInfo(storageDisks)
	for i, info := range disksInfo {
		if storageDisks[i] == nil {
			continue
		}
		msg += fmt.Sprintf(
			"\n[%s] %s - %s %s",
			int2Str(i+1, len(storageDisks)),
			storageDisks[i],
			humanize.IBytes(uint64(info.Total)),
			func() string {
				if info.Total > 0 {
					return "online"
				}
				return "offline"
			}(),
		)
	}
	return msg
}

// Prints regular message when we have sufficient disks to start the cluster.
func printRegularMsg(storageDisks []StorageAPI, fn printOnceFunc) {
	msg := getRegularMsg(storageDisks)
	fn(msg)
}

// Constructs a formatted regular message when we have sufficient disks to start the cluster.
func getRegularMsg(storageDisks []StorageAPI) string {
	msg := colorBlue("\nInitializing data volume.")
	disksInfo, _, _ := getDisksInfo(storageDisks)
	for i, info := range disksInfo {
		if storageDisks[i] == nil {
			continue
		}
		msg += fmt.Sprintf(
			"\n[%s] %s - %s %s",
			int2Str(i+1, len(storageDisks)),
			storageDisks[i],
			humanize.IBytes(uint64(info.Total)),
			func() string {
				if info.Total > 0 {
					return "online"
				}
				return "offline"
			}(),
		)
	}
	return msg
}

// Prints initialization message when cluster is being initialized for the first time.
func printFormatMsg(storageDisks []StorageAPI, fn printOnceFunc) {
	msg := getFormatMsg(storageDisks)
	fn(msg)
}

// Generate a formatted message when cluster is being initialized for the first time.
func getFormatMsg(storageDisks []StorageAPI) string {
	msg := colorBlue("\nInitializing data volume for the first time.")
	disksInfo, _, _ := getDisksInfo(storageDisks)
	for i, info := range disksInfo {
		if storageDisks[i] == nil {
			continue
		}
		msg += fmt.Sprintf(
			"\n[%s] %s - %s %s",
			int2Str(i+1, len(storageDisks)),
			storageDisks[i],
			humanize.IBytes(uint64(info.Total)),
			func() string {
				if info.Total > 0 {
					return "online"
				}
				return "offline"
			}(),
		)
	}
	return msg
}

func printConfigErrMsg(storageDisks []StorageAPI, sErrs []error, fn printOnceFunc) {
	msg := getConfigErrMsg(storageDisks, sErrs)
	fn(msg)
}

// Generate a formatted message when cluster is misconfigured.
func getConfigErrMsg(storageDisks []StorageAPI, sErrs []error) string {
	msg := colorBlue("\nDetected configuration inconsistencies in the cluster. Please fix following servers.")
	for i, disk := range storageDisks {
		if disk == nil {
			continue
		}
		if sErrs[i] == nil {
			continue
		}
		msg += fmt.Sprintf(
			"\n[%s] %s : %s",
			int2Str(i+1, len(storageDisks)),
			storageDisks[i],
			sErrs[i],
		)
	}
	return msg
}
