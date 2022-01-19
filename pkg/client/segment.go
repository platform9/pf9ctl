// Copyright © 2020 The Platform9 Systems Inc.

package client

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/platform9/pf9ctl/pkg/keystone"
	"github.com/platform9/pf9ctl/pkg/util"
	"go.uber.org/zap"
	"gopkg.in/segmentio/analytics-go.v3"
)

//Added segment key for the source PRD-PMKFT Metrics-Aggregator
var SegmentWriteKey string

type Segment interface {
	SendEvent(string, interface{}, string, string) error
	SendGroupTraits(string, interface{}) error
	Close()
}

type SegmentImpl struct {
	fqdn   string
	client analytics.Client
}

type NoopSegment struct {
}

type SegmentNoopLogger struct {
}

func NewSegment(fqdn string, noTracking bool) Segment {
	// mock out segment if the user wants no Tracking
	envCheck := os.Getenv("PF9CTL_SEGMENT_EVENTS_DISABLE")
	segmentEventDisabled, _ := strconv.ParseBool(envCheck)

	// Local build case
	if SegmentWriteKey == "" {
		segmentEventDisabled = true
	}

	if noTracking || segmentEventDisabled {
		return NoopSegment{}
	}
	client, _ := analytics.NewWithConfig(SegmentWriteKey, analytics.Config{
		Logger: &SegmentNoopLogger{},
	})

	return SegmentImpl{
		fqdn:   fqdn,
		client: client,
	}
}

func (c SegmentImpl) SendEvent(name string, data interface{}, status string, err string) error {
	//To differentiate between OVA and non-OVA node
	var infra string
	ovfservicepresent := InfraCheck()
	if ovfservicepresent {
		infra = "OVA"
	} else {
		infra = "CLI"
	}

	zap.S().Debug("Sending Segment Event: ", name)
	data_struct, ok := data.(keystone.KeystoneAuth)
	if ok {
		return c.client.Enqueue(analytics.Track{
			UserId: data_struct.UserID,
			Event:  name,
			Properties: analytics.NewProperties().
				Set("keystoneData", data).
				Set("dufqdn", data_struct.DUFqdn).
				Set("email", data_struct.Email).
				Set("status", status).
				Set("infra", infra).
				Set("errorMsg", err),
			Integrations: analytics.NewIntegrations().Set("Amplitude", map[string]interface{}{
				"session_id": time.Now().Unix(),
			}),
		})
	} else {
		return fmt.Errorf("Unable to fetch keystone info")
	}
}

func (c SegmentImpl) SendGroupTraits(name string, data interface{}) error {
	zap.S().Debug("Sending Group Trait: ", name)
	data_struct, ok := data.(keystone.KeystoneAuth)
	if ok {
		return c.client.Enqueue(analytics.Group{
			UserId:  data_struct.UserID,
			GroupId: name,
			Traits:  analytics.NewTraits().Set("data", data),
			Integrations: analytics.NewIntegrations().Set("Amplitude", map[string]interface{}{
				"session_id": time.Now().Unix(),
			}),
		})
	} else {
		return fmt.Errorf("Unable to fetch keystone info")
	}
}

func InfraCheck() bool {
	//Checking for OVF Service to determine infrastructure for node onboarding
	var ovfservice bool
	_, err1 := os.Stat(util.OVFLoc)
	if err1 != nil {
		zap.S().Debugf("OVF Service not present")
		ovfservice = false

	} else {
		zap.S().Debugf("Node onboarded through OVA")
		ovfservice = true
	}
	return ovfservice
}

func (c SegmentImpl) Close() {
	c.client.Close()
}

func (c *SegmentNoopLogger) Logf(format string, args ...interface{}) {
	zap.S().Debug("Could not send segment event")
}
func (c *SegmentNoopLogger) Errorf(format string, args ...interface{}) {
	zap.S().Debug("Could not send segment event")
}

// The Noop Implementation of Segment
func (c NoopSegment) SendEvent(name string, data interface{}, status string, err string) error {
	return nil
}

func (c NoopSegment) SendGroupTraits(name string, data interface{}) error {
	return nil
}

func (c NoopSegment) Close() {
	return
}
