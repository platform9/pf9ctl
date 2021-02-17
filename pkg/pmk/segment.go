// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"time"

	"github.com/google/uuid"
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
	if noTracking {
		return NoopSegment{}
	}
	client := analytics.New(segmentWriteKey)

	return SegmentImpl{
		fqdn:   fqdn,
		client: client,
	}
}

func (c SegmentImpl) SendEvent(name string, data interface{}) error {
	zap.S().Debug("Sending Segment Event: %s", name)
	return c.client.Enqueue(analytics.Track{
		AnonymousId: uuid.New().String(),
		Event:       name,
		Properties:  analytics.NewProperties().Set("data", data),
		Integrations: analytics.NewIntegrations().Set("Amplitude", map[string]interface{}{
			"session_id": time.Now().Unix(),
		}),
	})
}

func (c SegmentImpl) SendGroupTraits(name string, data interface{}) error {
	return c.client.Enqueue(analytics.Group{
		AnonymousId: uuid.New().String(),
		GroupId:     name,
		Traits:      analytics.NewTraits().Set("data", data),
		Integrations: analytics.NewIntegrations().Set("Amplitude", map[string]interface{}{
			"session_id": time.Now().Unix(),
		}),
	})
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
