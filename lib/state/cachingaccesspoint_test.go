/*
Copyright 2015 Gravitational, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

package state

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gravitational/teleport/lib/auth"
	"github.com/gravitational/teleport/lib/auth/testauthority"
	"github.com/gravitational/teleport/lib/backend/boltbk"
	"github.com/gravitational/teleport/lib/defaults"
	"github.com/gravitational/teleport/lib/services"
	"github.com/gravitational/teleport/lib/utils"

	"gopkg.in/check.v1"
)

// fake cluster we're testing on:
var (
	Nodes = []services.Server{
		{
			ID:        "1",
			Addr:      "10.50.0.1",
			Hostname:  "one",
			Labels:    make(map[string]string),
			CmdLabels: make(map[string]services.CommandLabel),
			Namespace: defaults.Namespace,
		},
		{
			ID:        "2",
			Addr:      "10.50.0.2",
			Hostname:  "two",
			Labels:    make(map[string]string),
			CmdLabels: make(map[string]services.CommandLabel),
			Namespace: defaults.Namespace,
		},
	}
	Proxies = []services.Server{
		{
			ID:       "3",
			Addr:     "10.50.0.3",
			Hostname: "three",
			Labels:   map[string]string{"os": "linux", "role": "proxy"},
			CmdLabels: map[string]services.CommandLabel{
				"uptime": {Period: time.Second, Command: []string{"uptime"}},
			},
		},
	}
	Users = []services.TeleportUser{
		{
			Name:           "elliot",
			AllowedLogins:  []string{"elliot", "root"},
			OIDCIdentities: []services.OIDCIdentity{},
		},
		{
			Name:          "bob",
			AllowedLogins: []string{"bob"},
			OIDCIdentities: []services.OIDCIdentity{
				{
					ConnectorID: "example.com",
					Email:       "bob@example.com",
				},
				{
					ConnectorID: "example.net",
					Email:       "bob@example.net",
				},
			},
		},
	}
)

type ClusterSnapshotSuite struct {
	dataDir    string
	backend    *boltbk.BoltBackend
	authServer *auth.AuthServer
}

var _ = check.Suite(&ClusterSnapshotSuite{})

// bootstrap check
func TestState(t *testing.T) { check.TestingT(t) }

func (s *ClusterSnapshotSuite) SetUpSuite(c *check.C) {
	utils.InitLoggerForTests()
}

func (s *ClusterSnapshotSuite) SetUpTest(c *check.C) {
	// create a new auth server:
	s.dataDir = c.MkDir()
	var err error
	s.backend, err = boltbk.New(filepath.Join(s.dataDir, "db"))
	c.Assert(err, check.IsNil)
	s.authServer = auth.NewAuthServer(&auth.InitConfig{
		Backend:    s.backend,
		Authority:  testauthority.New(),
		DomainName: "auth.local",
	})
	err = s.authServer.UpsertNamespace(
		services.NewNamespace(defaults.Namespace))
	c.Assert(err, check.IsNil)
	// add some nodes to it:
	for _, n := range Nodes {
		err = s.authServer.UpsertNode(n, defaults.ServerHeartbeatTTL)
		c.Assert(err, check.IsNil)
	}
	// add some proxies to it:
	for _, p := range Proxies {
		err = s.authServer.UpsertProxy(p, defaults.ServerHeartbeatTTL)
		c.Assert(err, check.IsNil)
	}
	// add some users to it:
	for _, u := range Users {
		err = s.authServer.UpsertUser(&u)
		c.Assert(err, check.IsNil)
	}
}

func (s *ClusterSnapshotSuite) TearDownTest(c *check.C) {
	s.authServer.Close()
	s.backend.Close()
	os.RemoveAll(s.dataDir)
}

func (s *ClusterSnapshotSuite) TestEverything(c *check.C) {
	snap, err := NewCachingAuthClient(s.authServer)
	c.Assert(err, check.IsNil)
	c.Assert(snap, check.NotNil)

	// kill the 'upstream' server:
	s.authServer.Close()

	users, err := snap.GetUsers()
	c.Assert(err, check.IsNil)
	c.Assert(users, check.HasLen, len(Users))

	nodes, err := snap.GetNodes(defaults.Namespace)
	c.Assert(err, check.IsNil)
	c.Assert(nodes, check.HasLen, len(Nodes))

	proxies, err := snap.GetProxies()
	c.Assert(err, check.IsNil)
	c.Assert(proxies, check.HasLen, len(Proxies))
}

func (s *ClusterSnapshotSuite) TestTry(c *check.C) {
	var (
		successfullCalls int
		failedCalls      int
	)
	success := func() error { successfullCalls++; return nil }
	failure := func() error { failedCalls++; return fmt.Errorf("eror") }

	ap, err := NewCachingAuthClient(s.authServer)
	c.Assert(err, check.IsNil)

	ap.try(success)
	ap.try(failure)

	c.Assert(successfullCalls, check.Equals, 1)
	c.Assert(failedCalls, check.Equals, 1)

	// these two calls should not happen because of a recent failure:
	ap.try(success)
	ap.try(failure)

	c.Assert(successfullCalls, check.Equals, 1)
	c.Assert(failedCalls, check.Equals, 1)

	// "wait" for backoff duration and try again:
	ap.lastErrorTime = time.Now().Add(-backoffDuration)

	ap.try(success)
	ap.try(failure)

	c.Assert(successfullCalls, check.Equals, 2)
	c.Assert(failedCalls, check.Equals, 2)
}
