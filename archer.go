package archers

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"time"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/host"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	leftHandler  = "/fromleft"
	rightHandler = "/fromright"
)

type Archer struct {
	start time.Time

	Node host.Host
	Prev *peer.ID
	Next *peer.ID
}

type Message struct {
	Data int `json:"data"`
}

func InitArchers(ctx context.Context, start time.Time, prevNode *peer.AddrInfo, amount int) (*Archer, error) {
	amount--
	node, err := libp2p.New(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "libp2p.New")
	}

	archer := &Archer{
		Node:  node,
		start: start,
	}

	nodeInfo := &peer.AddrInfo{
		ID:    node.ID(),
		Addrs: node.Addrs(),
	}

	if prevNode != nil {
		if err = archer.connectLeftNeighbor(ctx, *prevNode); err != nil {
			return nil, errors.Wrap(err, "archer.connectLeftNeighbor")
		}
	}
	if amount > 0 {
		nextNode, err := InitArchers(ctx, start, nodeInfo, amount)
		if err != nil {
			return nil, errors.Wrap(err, "InitArchers")
		}

		if err = archer.connectRightNeighbor(ctx, peer.AddrInfo{
			ID:    nextNode.Node.ID(),
			Addrs: nextNode.Node.Addrs(),
		}); err != nil {
			return nil, errors.Wrap(err, "archer.connectRightNeighbor")
		}
	}

	return archer, nil
}

func (a *Archer) Start(ctx context.Context) error {
	count := 1
	if err := a.messageToTheRight(ctx, &Message{Data: count}); err != nil {
		return err
	}
	return nil
}

func (a *Archer) fireWithTimeout(timeout int) {
	time.Sleep(time.Second * time.Duration(timeout))
	fmt.Println("FIRE ", a.Node.ID(), time.Now().Sub(a.start).Seconds())
}

func (a *Archer) connectLeftNeighbor(ctx context.Context, n peer.AddrInfo) error {
	if err := a.Node.Connect(ctx, n); err != nil {
		return errors.Wrapf(err, "node.Connect <%s>", n.String())
	}
	a.Prev = &n.ID
	a.Node.SetStreamHandler(leftHandler, a.fromLeftHandler)
	return nil
}

func (a *Archer) connectRightNeighbor(ctx context.Context, n peer.AddrInfo) error {
	if err := a.Node.Connect(ctx, n); err != nil {
		return errors.Wrapf(err, "node.Connect <%s>", n.String())
	}
	a.Next = &n.ID
	a.Node.SetStreamHandler(rightHandler, a.fromRightHandler)
	return nil
}
func (a *Archer) fromLeftHandler(s network.Stream) {
	logrus.Info("listener received new message from left")
	ctx := context.Background()

	m, err := read(s)
	if err != nil {
		logrus.Errorf("fromLeftHandler archer %s err: %v", a.Node.ID(), err)
	}
	m.Data++
	if a.Next != nil {
		if err = a.messageToTheRight(ctx, m); err != nil {
			logrus.Errorf("fromLeftHandler archer %s messageToTheRight err: %v", a.Node.ID(), err)
		}
	} else {
		if err = a.messageToTheLeft(ctx, m); err != nil {
			logrus.Errorf("fromLeftHandler archer %s messageToTheLeft err: %v", a.Node.ID(), err)
		}
		a.fireWithTimeout(m.Data)
	}
}

func (a *Archer) fromRightHandler(s network.Stream) {
	logrus.Info("listener received new message from right")
	ctx := context.Background()

	m, err := read(s)
	if err != nil {
		logrus.Errorf("fromRightHandler archer %s err: %v", a.Node.ID(), err)
	}
	if a.Prev != nil {
		m.Data--
		if err = a.messageToTheLeft(ctx, m); err != nil {
			logrus.Errorf("fromLeftHandler archer %s messageToTheLeft err: %v", a.Node.ID(), err)
		}
	}
	a.fireWithTimeout(m.Data)
}

func (a *Archer) messageToTheLeft(ctx context.Context, m *Message) error {
	s, err := a.Node.NewStream(ctx, *a.Prev, rightHandler)
	if err != nil {
		return errors.Wrap(err, "node.NewStream")
	}
	return a.sendMessage(s, m)
}

func (a *Archer) messageToTheRight(ctx context.Context, m *Message) error {
	s, err := a.Node.NewStream(ctx, *a.Next, leftHandler)
	if err != nil {
		return errors.Wrap(err, "node.NewStream")
	}
	return a.sendMessage(s, m)
}

func (a *Archer) sendMessage(s network.Stream, m *Message) error {
	time.Sleep(time.Second)
	b, err := json.Marshal(m)
	if err != nil {
		return errors.Wrap(err, "json.Marshal")
	}
	_, err = s.Write(b)
	if err != nil {
		return errors.Wrap(err, "s.Write")
	}
	s.Close()
	return nil
}

func read(s network.Stream) (*Message, error) {
	m := &Message{}
	defer s.Close()
	b, err := ioutil.ReadAll(s)
	if err != nil {
		return m, err
	}
	if err = json.Unmarshal(b, m); err != nil {
		return m, err
	}
	log.Printf("read: %d", m.Data)
	return m, err
}
