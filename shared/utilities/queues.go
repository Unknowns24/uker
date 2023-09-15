package uker

// Global interface
type Queues interface {
	CreateConsumer() error
	CreateProducer() error
}

// Local struct to be implmented
type queue struct{}

// External contructor
func NewQueue() Queues {
	return &queue{}
}

// CreateConsumer function implementation
func (q *queue) CreateConsumer() error {
	//TODO: Implement..
	return nil
}

// CreateProducer function implementation
func (q *queue) CreateProducer() error {
	//TODO: Implement..
	return nil
}
