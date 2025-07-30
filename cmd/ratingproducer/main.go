package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/abhishek622/movieapp/rating/pkg/model"
	"github.com/confluentinc/confluent-kafka-go/kafka"
)

func main() {
	fmt.Println("Creating a kafka producer")

	producer, err := kafka.NewProducer(&kafka.ConfigMap{
		"bootstrap.servers": "localhost:9092", // 1️⃣ add port
	})
	if err != nil {
		log.Fatalf("cannot create producer: %v", err)
	}
	defer producer.Close()

	// 2️⃣ delivery-report goroutine
	go func() {
		for e := range producer.Events() {
			switch ev := e.(type) {
			case *kafka.Message:
				if ev.TopicPartition.Error != nil {
					log.Printf("delivery failed: %v", ev.TopicPartition)
				}
			}
		}
	}()

	const fileName = "ratingsdata.json"
	fmt.Println("Reading rating events from file " + fileName)

	ratingEvents, err := readRatingEvents(fileName)
	if err != nil {
		log.Fatalf("cannot read events: %v", err)
	}

	const topic = "ratings"
	if err := produceRatingEvents(topic, producer, ratingEvents); err != nil {
		log.Fatalf("cannot produce events: %v", err)
	}

	// 3️⃣ wait for outstanding messages or timeout
	remaining := producer.Flush(10_000) // 10 s
	if remaining != 0 {
		log.Fatalf("still %d messages not delivered", remaining)
	}
	fmt.Println("all events produced")
}

func readRatingEvents(fileName string) ([]model.RatingEvent, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var ratings []model.RatingEvent
	if err := json.NewDecoder(f).Decode(&ratings); err != nil {
		return nil, err
	}
	return ratings, nil
}

func produceRatingEvents(topic string, producer *kafka.Producer, events []model.RatingEvent) error {
	for _, re := range events {
		payload, err := json.Marshal(re)
		if err != nil {
			return err
		}
		if err := producer.Produce(&kafka.Message{
			TopicPartition: kafka.TopicPartition{Topic: &topic, Partition: kafka.PartitionAny},
			Value:          payload,
		}, nil); err != nil {
			return err
		}
	}
	return nil
}
