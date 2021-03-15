// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/platform9/pf9ctl/pkg/keystone"
	"go.uber.org/zap"
	"gopkg.in/segmentio/analytics-go.v3"
)

const segmentWriteKey = "P6DycMCALprZrUwWL9ZzRLlfMQwL5Xyl"

type Segment interface {
	SendEvent(string, interface{}) error
	SendGroupTraits(string, interface{}) error
	Close()
}

type SegmentImpl struct {
	fqdn   string
	client analytics.Client
}

type NoopSegment struct {
}

func NewSegment(fqdn string, noTracking bool) Segment {
	// mock out segment if the user wants no Tracking
	envCheck := os.Getenv("PF9CTL_SEGMENT_EVENTS_DISABLE")
	segmentEventDisabled, _ := strconv.ParseBool(envCheck)
	if noTracking || segmentEventDisabled {
		return NoopSegment{}
	}
	client := analytics.New(segmentWriteKey)

	return SegmentImpl{
		fqdn:   fqdn,
		client: client,
	}
}

func (c SegmentImpl) SendEvent(name string, data interface{}) error {
	zap.S().Debug("Sending Segment Event: ", name)
	data_struct, ok := data.(keystone.KeystoneAuth)
	if ok {
		zap.S().Debug("Sending Segment data: ", data_struct.UserID)
		return c.client.Enqueue(analytics.Track{
			UserId:     data_struct.UserID,
			Event:      name,
			Properties: analytics.NewProperties().Set("data", data),
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
		zap.S().Debug("Sending Segment data: ", data_struct.UserID)
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

func (c SegmentImpl) Close() {
	c.client.Close()
}

// The Noop Implementation of Segment
func (c NoopSegment) SendEvent(name string, data interface{}) error {
	return nil
}

func (c NoopSegment) SendGroupTraits(name string, data interface{}) error {
	return nil
}

func (c NoopSegment) Close() {
	return
}
