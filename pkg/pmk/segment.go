// Copyright © 2020 The Platform9 Systems Inc.

package pmk

import (
	"time"

	"github.com/google/uuid"
	"gopkg.in/segmentio/analytics-go.v3"
)

const WriteKey = "P6DycMCALprZrUwWL9ZzRLlfMQwL5Xyl"

type Segment interface {
	SendEvent(string, interface{}) error
	SendGroupTraits(string, interface{}) error
	Close()
}

type SegmentImpl struct {
	fqdn   string
	client analytics.Client
}

func NewSegment(fqdn string) Segment {
	client := analytics.New(WriteKey)

	return SegmentImpl{
		fqdn:   fqdn,
		client: client,
	}
}

func (c SegmentImpl) SendEvent(name string, data interface{}) error {
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
