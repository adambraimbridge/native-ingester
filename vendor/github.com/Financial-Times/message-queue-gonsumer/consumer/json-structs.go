package consumer

//QueueConfig represents the configuration of the queue, consumer group and topic the consumer interested about.
type QueueConfig struct {
	Addrs                []string `json:"address"` //list of queue addresses.
	Group                string   `json:"group"`
	Topic                string   `json:"topic"`
	Queue                string   `json:"queue"` //The name of the queue. Leave it empty for requests to UCS kafka-proxy
	Offset               string   `json:"offset"`
	BackoffPeriod        int      `json:"backoffPeriod"`
	StreamCount          int      `json:"streamCount"`
	ConcurrentProcessing bool     `json:"concurrentProcessing"`
	AuthorizationKey     string
	AutoCommitEnable     bool `json:"autoCommitEnable"`
	NoOfProcessors       int
}

type consumer struct {
	BaseURI    string `json:"base_uri"`
	InstanceID string `json:",instance_id"`
}
