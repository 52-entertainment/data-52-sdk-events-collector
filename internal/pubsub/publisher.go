package pubsub

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	gcpubsub "cloud.google.com/go/pubsub/v2"
)

type Publisher interface {
	Publish(ctx context.Context, data []byte, attrs map[string]string) error
	Close() error
}

type PubSubPublisher struct {
	client    *gcpubsub.Client
	publisher *gcpubsub.Publisher
}

func NewPublisher(
	ctx context.Context,
	projectID, topic string,
) (*PubSubPublisher, error) {
	if projectID == "" {
		return nil, errors.New("empty projectID")
	}
	if topic == "" {
		return nil, errors.New("empty topic")
	}

	topicName := topic
	if !strings.HasPrefix(topicName, "projects/") {
		topicName = fmt.Sprintf("projects/%s/topics/%s", projectID, topic)
	}

	client, err := gcpubsub.NewClient(ctx, projectID)
	if err != nil {
		return nil, err
	}

	publisher := client.Publisher(topicName)

	// batching defaults.
	publisher.PublishSettings.ByteThreshold = 1 << 20 // 1 MiB
	publisher.PublishSettings.DelayThreshold = 50 * time.Millisecond
	publisher.PublishSettings.CountThreshold = 100

	return &PubSubPublisher{
		client:    client,
		publisher: publisher,
	}, nil
}

func (p *PubSubPublisher) Publish(
	ctx context.Context,
	data []byte,
	attrs map[string]string,
) error {
	res := p.publisher.Publish(ctx, &gcpubsub.Message{
		Data:       data,
		Attributes: attrs,
	})

	_, err := res.Get(ctx)
	return err
}

func (p *PubSubPublisher) Close() error {
	// Stops background goroutines created by Publish().
	p.publisher.Stop()
	return p.client.Close()
}
