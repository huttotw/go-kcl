# go-kcl

# Introduction
This package is a simple and somewhat naïve client library for Kinesis. It was developed in order to provide an easy way to read from a Kinesis stream. We simply get the shards for the stream, poll each shard for new records, and return them to a handler function, from which you can do whatever you want.

# Example
```go
func main() {
	sess, err := session.NewSession()
	if err != nil {
		panic(err)
	}

	s := kcl.NewLocalStore()
	config := kcl.Config{
		Limit:        1000,
		Interval:     time.Millisecond * 1000,
		IteratorType: kcl.IteratorTypeLatest,
	}
	k, err := kcl.NewStream(sess, os.Getenv("AWS_KINESIS_STREAM"), s, config)
	if err != nil {
		panic(err)
	}

	err = k.Listen(handler)
	if err != nil {
		panic(err)
	}
}

func handler(records []*kinesis.Record) {
	for _, r := range records {
		fmt.Println(*r.SequenceNumber)
	}
}
```

# Understanding Kinesis
* What interval is appropriate for my stream?
* What is the maximum number of records I should return?
* How will I store the iterator? What if I am running this library in a distributed fashion?

### Interval
This library works similar to Lambda with Kinesis. We simply poll the stream at every interval, and attempt to get the maximum number of records each time. You should understand the limits of reading from Kinesis by reading these [docs](http://docs.aws.amazon.com/streams/latest/dev/service-sizes-and-limits.html).

Each shard can only be read 5 times per second. This means that if you have this package running in a distributed fashion, you could run into limits.

### Limit
Kinesis has a limit of 2MB per second, you should consider your record size when configuring the limit for this package. For more information, check out [Kinesis' limits](http://docs.aws.amazon.com/streams/latest/dev/service-sizes-and-limits.html).

### Storing the iterator
Kinesis keeps track of your place on the stream by using an iterator. An iterator is simply a string that denotes which record you left off on. Initially, this package makes a request to Kinesis in order to get the place in the stream. Each time we get more records, a new iterator is returned.

It is important to have some record of this in persitent storage in case your application crashes.

If you are running in a distributed fashion, your store should be safe for concurrent use.

**Iterator Types**
* **LATEST** - you will start with the next record that is put onto the stream.
* **TRIM_HORIZON** - you will start with the oldest record on the stream, and work towards the head.
* **AT_SEQUENCE_NUMBER** - you will start at the given sequence number. Sequence numbers are sequential since the beginning of time for each shard.
* **AFTER_SEQUENCE_NUMBER** - similar to At Sequence Number, but after.
* **AT_TIMESTAMP** - you will start at the first record at a given timestamp and work towards the head.

_The iterator type only matters for the first time you pull records, after that, you will get records in order while working towards the head._

# License
Licensed under the Apache License, Version 2.0 (the "License"); you may not use this work except in compliance with the License. You may obtain a copy of the License in the LICENSE file, or at:

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.