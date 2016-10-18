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
	"sync"
	"time"
)

// SystemLockState - Structure to fill the lock state of entire object storage.
// That is the total locks held, total calls blocked on locks and state of all the locks for the entire system.
type SystemLockState struct {
	TotalLocks int64 `json:"totalLocks"`
	// Count of operations which are blocked waiting for the lock to
	// be released.
	TotalBlockedLocks int64 `json:"totalBlockedLocks"`
	// Count of operations which has successfully acquired the lock but
	// hasn't unlocked yet( operation in progress).
	TotalAcquiredLocks int64            `json:"totalAcquiredLocks"`
	LocksInfoPerObject []VolumeLockInfo `json:"locksInfoPerObject"`
}

// VolumeLockInfo - Structure to contain the lock state info for volume, path pair.
type VolumeLockInfo struct {
	Bucket string `json:"bucket"`
	Object string `json:"object"`
	// All locks blocked + running for given <volume,path> pair.
	LocksOnObject int64 `json:"locksOnObject"`
	// Count of operations which has successfully acquired the lock
	// but hasn't unlocked yet( operation in progress).
	LocksAcquiredOnObject int64 `json:"locksAcquiredOnObject"`
	// Count of operations which are blocked waiting for the lock
	// to be released.
	TotalBlockedLocks int64 `json:"locksBlockedOnObject"`
	// State information containing state of the locks for all operations
	// on given <volume,path> pair.
	LockDetailsOnObject []OpsLockState `json:"lockDetailsOnObject"`
}

// OpsLockState - structure to fill in state information of the lock.
// structure to fill in status information for each operation with given operation ID.
type OpsLockState struct {
	OperationID string        `json:"opsID"`          // String containing operation ID.
	LockOrigin  string        `json:"lockOrigin"`     // Operation type (GetObject, PutObject...)
	LockType    lockType      `json:"lockType"`       // Lock type (RLock, WLock)
	Status      statusType    `json:"status"`         // Status can be Running/Ready/Blocked.
	Since       time.Time     `json:"statusSince"`    // Time when the lock was initially held.
	Duration    time.Duration `json:"statusDuration"` // Duration since the lock was held.
}

// Read entire state of the locks in the system and return.
func getSystemLockState() (SystemLockState, error) {
	nsMutex.lockMapMutex.Lock()
	defer nsMutex.lockMapMutex.Unlock()

	lockState := SystemLockState{}

	lockState.TotalBlockedLocks = nsMutex.blockedCounter
	lockState.TotalLocks = nsMutex.globalLockCounter
	lockState.TotalAcquiredLocks = nsMutex.runningLockCounter

	for param, debugLock := range nsMutex.debugLockMap {
		volLockInfo := VolumeLockInfo{}
		volLockInfo.Bucket = param.volume
		volLockInfo.Object = param.path
		volLockInfo.LocksOnObject = debugLock.ref
		volLockInfo.TotalBlockedLocks = debugLock.blocked
		volLockInfo.LocksAcquiredOnObject = debugLock.running
		for opsID, lockInfo := range debugLock.lockInfo {
			volLockInfo.LockDetailsOnObject = append(volLockInfo.LockDetailsOnObject, OpsLockState{
				OperationID: opsID,
				LockOrigin:  lockInfo.lockOrigin,
				LockType:    lockInfo.lType,
				Status:      lockInfo.status,
				Since:       lockInfo.since,
				Duration:    time.Now().UTC().Sub(lockInfo.since),
			})
		}
		lockState.LocksInfoPerObject = append(lockState.LocksInfoPerObject, volLockInfo)
	}
	return lockState, nil
}

// Remote procedure call, calls LockInfo handler with given input args.
func (c *controlAPIHandlers) remoteLockInfoCall(args *GenericArgs, replies []SystemLockState) error {
	var wg sync.WaitGroup
	var errs = make([]error, len(c.RemoteControls))
	// Send remote call to all neighboring peers to restart minio servers.
	for index, clnt := range c.RemoteControls {
		wg.Add(1)
		go func(index int, client *AuthRPCClient) {
			defer wg.Done()
			errs[index] = client.Call("Control.RemoteLockInfo", args, &replies[index])
			errorIf(errs[index], "Unable to initiate control lockInfo request to remote node %s", client.Node())
		}(index, clnt)
	}
	wg.Wait()
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}

// RemoteLockInfo - RPC control handler for `minio control lock`, used internally by LockInfo to
// make calls to neighboring peers.
func (c *controlAPIHandlers) RemoteLockInfo(args *GenericArgs, reply *SystemLockState) error {
	if !isRPCTokenValid(args.Token) {
		return errInvalidToken
	}
	// Obtain the lock state information of the local system.
	lockState, err := getSystemLockState()
	// In case of error, return err to the RPC client.
	if err != nil {
		return err
	}
	*reply = lockState
	return nil
}

// LockInfo - RPC control handler for `minio control lock`. Returns the info of the locks held in the cluster.
func (c *controlAPIHandlers) LockInfo(args *GenericArgs, reply *map[string]SystemLockState) error {
	if !isRPCTokenValid(args.Token) {
		return errInvalidToken
	}
	var replies = make([]SystemLockState, len(c.RemoteControls))
	if args.Remote {
		// Fetch lock states from all the remote peers.
		args.Remote = false
		if err := c.remoteLockInfoCall(args, replies); err != nil {
			return err
		}
	}
	rep := make(map[string]SystemLockState)
	// The response containing the lock info.
	for index, client := range c.RemoteControls {
		rep[client.Node()] = replies[index]
	}
	// Obtain the lock state information of the local system.
	lockState, err := getSystemLockState()
	// In case of error, return err to the RPC client.
	if err != nil {
		return err
	}

	// Save the local node lock state.
	rep[c.LocalNode] = lockState

	// Set the reply.
	*reply = rep

	// Success.
	return nil
}
