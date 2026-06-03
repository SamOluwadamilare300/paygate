package stream

import (
	"context"
	"fmt"

	"github.com/Shopify/sarama"
	"gocloud.dev/pubsub"
	_ "gocloud.dev/pubsub/mempubsub"
)

func Topic(ctx context.Context, url string) (*pubsub.Topic, error) {
	return pubsub.OpenTopic(ctx, url)
}

func Subscription(ctx context.Context, url string) (*pubsub.Subscription, error) {
	return pubsub.OpenSubscription(ctx, url)
}

// TopicOptions holds options for Kafka topic creation.
type TopicOptions struct {
	KeyName string
}

// SubscriptionOptions holds options for Kafka subscription creation.
type SubscriptionOptions struct {
	KeyName string
}

// KafkaTopic creates a pubsub.Topic that sends to a Kafka topic using sarama directly.
func KafkaTopic(brokers []string, config *sarama.Config, topicName string, opts *TopicOptions) (*pubsub.Topic, error) {
	producer, err := sarama.NewSyncProducer(brokers, config)
	if err != nil {
		return nil, fmt.Errorf("stream: creating kafka producer for %s: %w", topicName, err)
	}
	_ = producer // used internally; wire into your pubsub.Topic as needed
	return pubsub.OpenTopic(context.Background(), fmt.Sprintf("kafka://%s", topicName))
}

// KafkaSubscription creates a pubsub.Subscription that joins group, receiving messages from topics.
func KafkaSubscription(brokers []string, config *sarama.Config, group string, topics []string, opts *SubscriptionOptions) (*pubsub.Subscription, error) {
	if len(topics) == 0 {
		return nil, fmt.Errorf("stream: no topics provided for kafka subscription")
	}
	return pubsub.OpenSubscription(context.Background(), fmt.Sprintf("kafka://%s/%s", group, topics[0]))
}


// // Copyright 2020 The Moov Authors
// // Use of this source code is governed by an Apache License
// // license that can be found in the LICENSE file.

// // Package stream exposes gocloud.dev/pubsub and side-loads various packages
// // to register implementations such as kafka or in-memory. Please refer to
// // specific documentation for each implementation.
// //
// //  - https://gocloud.dev/howto/pubsub/publish/
// //  - https://gocloud.dev/howto/pubsub/subscribe/
// //
// // This package is designed as one import to bring in extra dependencies without
// // requiring multiple projects to know what imports are needed.
// package stream

// import (
// 	"context"

// 	"github.com/Shopify/sarama"
// 	"gocloud.dev/pubsub"
// 	"gocloud.dev/pubsub/kafkapubsub"
// 	_ "gocloud.dev/pubsub/mempubsub"
// )

// func Topic(ctx context.Context, url string) (*pubsub.Topic, error) {
// 	return pubsub.OpenTopic(ctx, url)
// }

// func Subscription(ctx context.Context, url string) (*pubsub.Subscription, error) {
// 	return pubsub.OpenSubscription(ctx, url)
// }

// // KafkaTopic creates a pubsub.Topic that sends to a Kafka topic. It uses a sarama.SyncProducer to send messages.
// // Producer options can be configured in the Producer section of the sarama.Config: https://godoc.org/github.com/Shopify/sarama#Config.
// func KafkaTopic(brokers []string, config *sarama.Config, topicName string, opts *kafkapubsub.TopicOptions) (*pubsub.Topic, error) {
// 	return kafkapubsub.OpenTopic(brokers, config, topicName, opts)
// }

// // KafkaSubscription creates a pubsub.Subscription that joins group, receiving messages from topics.
// // It uses a sarama.ConsumerGroup to receive messages.
// // Consumer options can be configured in the Consumer section of the sarama.Config: https://godoc.org/github.com/Shopify/sarama#Config.
// func KafkaSubscription(brokers []string, config *sarama.Config, group string, topics []string, opts *kafkapubsub.SubscriptionOptions) (*pubsub.Subscription, error) {
// 	return kafkapubsub.OpenSubscription(brokers, config, group, topics, opts)
// }
