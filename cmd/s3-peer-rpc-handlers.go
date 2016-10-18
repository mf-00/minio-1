/*
 * Minio Cloud Storage, (C) 2014-2016 Minio, Inc.
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
	"encoding/json"
	"time"
)

func (s3 *s3PeerAPIHandlers) LoginHandler(args *RPCLoginArgs, reply *RPCLoginReply) error {
	jwt, err := newJWT(defaultInterNodeJWTExpiry)
	if err != nil {
		return err
	}
	if err = jwt.Authenticate(args.Username, args.Password); err != nil {
		return err
	}
	token, err := jwt.GenerateToken(args.Username)
	if err != nil {
		return err
	}
	reply.Token = token
	reply.ServerVersion = Version
	reply.Timestamp = time.Now().UTC()
	return nil
}

// SetBNPArgs - Arguments collection to SetBucketNotificationPeer RPC
// call
type SetBNPArgs struct {
	// For Auth
	GenericArgs

	Bucket string

	// Notification config for the given bucket.
	NCfg *notificationConfig
}

func (s3 *s3PeerAPIHandlers) SetBucketNotificationPeer(args *SetBNPArgs, reply *GenericReply) error {
	// check auth
	if !isRPCTokenValid(args.Token) {
		return errInvalidToken
	}

	// check if object layer is available.
	objAPI := s3.ObjectAPI()
	if objAPI == nil {
		return errServerNotInitialized
	}

	// Update in-memory notification config.
	globalEventNotifier.SetBucketNotificationConfig(args.Bucket, args.NCfg)

	return nil
}

// SetBLPArgs - Arguments collection to SetBucketListenerPeer RPC call
type SetBLPArgs struct {
	// For Auth
	GenericArgs

	Bucket string

	// Listener config for a given bucket.
	LCfg []listenerConfig
}

func (s3 *s3PeerAPIHandlers) SetBucketListenerPeer(args SetBLPArgs, reply *GenericReply) error {
	// check auth
	if !isRPCTokenValid(args.Token) {
		return errInvalidToken
	}

	// check if object layer is available.
	objAPI := s3.ObjectAPI()
	if objAPI == nil {
		return errServerNotInitialized
	}

	// Update in-memory notification config.
	return globalEventNotifier.SetBucketListenerConfig(args.Bucket, args.LCfg)
}

// EventArgs - Arguments collection for Event RPC call
type EventArgs struct {
	// For Auth
	GenericArgs

	// event being sent
	Event []NotificationEvent

	// client that it is meant for
	Arn string
}

// submit an event to the receiving server.
func (s3 *s3PeerAPIHandlers) Event(args *EventArgs, reply *GenericReply) error {
	// check auth
	if !isRPCTokenValid(args.Token) {
		return errInvalidToken
	}

	// check if object layer is available.
	objAPI := s3.ObjectAPI()
	if objAPI == nil {
		return errServerNotInitialized
	}

	return globalEventNotifier.SendListenerEvent(args.Arn, args.Event)
}

// SetBPPArgs - Arguments collection for SetBucketPolicyPeer RPC call
type SetBPPArgs struct {
	// For Auth
	GenericArgs

	Bucket string

	// Policy change (serialized to JSON)
	PChBytes []byte
}

// tell receiving server to update a bucket policy
func (s3 *s3PeerAPIHandlers) SetBucketPolicyPeer(args SetBPPArgs, reply *GenericReply) error {
	// check auth
	if !isRPCTokenValid(args.Token) {
		return errInvalidToken
	}

	// check if object layer is available.
	objAPI := s3.ObjectAPI()
	if objAPI == nil {
		return errServerNotInitialized
	}

	var pCh policyChange
	if err := json.Unmarshal(args.PChBytes, &pCh); err != nil {
		return err
	}

	return globalBucketPolicies.SetBucketPolicy(args.Bucket, pCh)
}
