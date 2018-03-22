package kcl

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kinesis"
)

// Define our iterator types, see https://docs.aws.amazon.com/sdk-for-go/api/service/kinesis/#GetShardIteratorInput
// for more detail.
const (
	IteratorTypeAtSequenceNumber    = "AT_SEQUENCE_NUMBER"
	IteratorTypeAfterSequenceNumber = "AFTER_SEQUENCE_NUMER"
	IteratorTypeAtTimestamp         = "AT_TIMESTAMP"
	IteratorTypeTrimHorizon         = "TRIM_HORIZON"
	IteratorTypeLatest              = "LATEST"
)

// Stream will keep track of where we are at on the stream for each shard
type Stream struct {
	// Shards are all the shards that belong to the stream
	Shards []Shard

	// Logger is an interface that can be used to debug your stream
	Logger Logger

	// Config defines how we will interact with the underlying stream
	config Config

	// Name is the name of the stream
	Name string

	// The Kinesis service that we will use to make calls to AWS
	svc *kinesis.Kinesis

	// The store that we use to store the latest iterator
	store Store
}

// Config sets some properties that affect how we interact with the Kinesis
// stream.
type Config struct {
	// The amount of time in between each GetRecords request. In order to not
	// exceed your ReadThroughput, you should consider the number of concurrent
	// consumers you have running.
	Interval time.Duration

	// IteratorType is the type of iterator that we want to use to read from the stream.
	// This denotes our starting position in the stream.
	IteratorType string

	// The maximum amount of records we will get on one GetRecords request.
	// In order to not run into Kinesis limits, you should consider the size
	// of your records.
	Limit int64
}

// Shard is a shard on the Kinesis stream
type Shard struct {
	// The identifier of the shard inside the stream
	ID string

	// The sequence number to start at
	StartAt string
}

// HandlerFunc is the argument to the listen function, for every batch of records that comes
// off of the Kinesis stream, we will call the HandlerFunc once.
type HandlerFunc func(records []*kinesis.Record)

// NewStream will return a pointer to a stream that you can listen on. Stream is capable of
// managing multiple shards, printing out log statements, and polling Kinesis at a regular
// interval.
func NewStream(sess *session.Session, kinesisEndpoint string, stream string, store Store, config Config) (*Stream, error) {
	svc := kinesis.New(sess, &aws.Config{Endpoint: aws.String(kinesisEndpoint)})
	resp, err := svc.DescribeStream(&kinesis.DescribeStreamInput{
		StreamName: aws.String(stream),
	})
	if err != nil {
		return nil, err
	}

	var shards = make([]Shard, 0)
	for _, shard := range resp.StreamDescription.Shards {
		s := Shard{
			ID:      aws.StringValue(shard.ShardId),
			StartAt: aws.StringValue(shard.SequenceNumberRange.StartingSequenceNumber),
		}
		shards = append(shards, s)
	}

	if len(shards) == 0 {
		return nil, fmt.Errorf("kcl: stream has 0 shards")
	}

	s := Stream{
		Shards: shards,
		Logger: noOpLogger{},
		config: config,
		Name:   stream,
		svc:    svc,
		store:  store,
	}

	return &s, nil
}

// Listen will call the HandlerFunc for each batch of events that come off the Kinesis stream.
// Listen will poll the Kinesis Stream every interval, and handle any new records. We use the
// store to keep track of our position in the stream so that we avoid reading recoreds twice,
// or not progressing in the stream.
func (s *Stream) Listen(handler HandlerFunc) error {
	tick := time.NewTicker(s.config.Interval).C

	// Set the starting position for each shard
	err := setInitialIterators(s)
	if err != nil {
		return err
	}

	// Start listening
	for {
		select {
		case <-tick:
			s.Logger.Log("level", "info", "msg", "tick")
			for _, shard := range s.Shards {
				iterator, err := s.store.GetShardIterator(s.Name, shard.ID)
				if err != nil {
					return err
				}

				s.Logger.Log("level", "info", "msg", "getting records for shard", "shard", shard.ID, "iterator", iterator)
				resp, err := s.svc.GetRecords(&kinesis.GetRecordsInput{
					Limit:         aws.Int64(s.config.Limit),
					ShardIterator: aws.String(iterator),
				})
				if err != nil {
					return err
				}

				s.Logger.Log("level", "info", "msg", "passing records to handler function")
				go handler(resp.Records)

				err = s.store.UpdateShardIterator(s.Name, shard.ID, aws.StringValue(resp.NextShardIterator))
				if err != nil {
					return err
				}
			}
		}
	}
}

// setInitialIterators will find the starting position for all shards based on the
// iterator type given in the config
func setInitialIterators(s *Stream) error {
	// Get the initial position of all the shards
	s.Logger.Log("level", "info", "msg", "getting initial shard iterators for all shards")
	for _, shard := range s.Shards {
		resp, err := s.svc.GetShardIterator(&kinesis.GetShardIteratorInput{
			ShardId:           aws.String(shard.ID),
			ShardIteratorType: aws.String(s.config.IteratorType),
			StreamName:        aws.String(s.Name),
		})
		if err != nil {
			return err
		}
		err = s.store.UpdateShardIterator(s.Name, shard.ID, aws.StringValue(resp.ShardIterator))
		if err != nil {
			return err
		}
	}

	return nil
}
