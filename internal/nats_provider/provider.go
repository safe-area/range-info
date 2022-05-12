package nats_provider

import (
	"errors"
	"github.com/nats-io/nats.go"
	"github.com/sirupsen/logrus"
	"strings"
	"time"
)

var (
	// ErrAlreadySubscribed is an error firing when caller tries to subscribe to the same subject they are already subscribed
	ErrAlreadySubscribed = errors.New("only one subscription per subject supported")
)

// NATSProvider is a struct representing NATS communications.
type NATSProvider struct {
	conn           *nats.Conn
	subs           map[string]*nats.Subscription
	connectionURLs []string
}

// New is the constructor for NATSProvider.
// connectionURLs - addresses of NATS cluster.
// logger - logger to log error events into.
func New(connectionURLs []string) *NATSProvider {
	return &NATSProvider{
		connectionURLs: connectionURLs,
		subs:           make(map[string]*nats.Subscription),
	}
}

// Open connects to NATS cluster by addresses provided in constructor.
func (p *NATSProvider) Open() (err error) {
	p.conn, err = nats.Connect(
		strings.Join(p.connectionURLs, ","),
		nats.Timeout(5*time.Second),
		nats.DisconnectErrHandler(func(conn *nats.Conn, err error) {
			if err != nil {
				logrus.Errorf("nats disconnect: %s", err)
			}
		}),
		nats.ErrorHandler(func(conn *nats.Conn, s *nats.Subscription, err error) {
			logrus.Errorf("nats error: %s", err)
		}),
	)
	return err
}

// Close closes the connection with NATS server/cluster.
func (p *NATSProvider) Close() {
	p.conn.Close()
}

// Subscribe creates a subscription to specified subject with the provided callback.
// Only one subscruption per subject supported, to change the subscription unsubscribe first.
// Attempting to subscribe for the same subject will return ErrAlreadySubscribed, the subscription stays unchanged.
func (p *NATSProvider) Subscribe(subject string, callback func(*nats.Msg)) error {
	if _, ok := p.subs[subject]; ok {
		return ErrAlreadySubscribed
	}
	sub, err := p.conn.Subscribe(subject, callback)
	if err != nil {
		return err
	}
	p.subs[subject] = sub
	return nil
}

// Unsubscribe unsubscribes from the provided subject. If no subscription exists for the subject, it's a no-op.
func (p *NATSProvider) Unsubscribe(subject string) error {
	sub, ok := p.subs[subject]
	if ok {
		err := sub.Unsubscribe()
		if err != nil {
			return err
		}
		delete(p.subs, subject)
	}
	return nil
}

// Publish publishes a message with contents in msg into the provided subject.
func (p *NATSProvider) Publish(subject string, msg []byte) error {
	return p.conn.Publish(subject, msg)
}

func (p *NATSProvider) Request(subject string, msg []byte) (*nats.Msg, error) {
	return p.conn.Request(subject, msg, 5*time.Second)
}

func (p *NATSProvider) Flush() error {
	return p.conn.Flush()
}
