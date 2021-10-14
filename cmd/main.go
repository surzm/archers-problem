package main

import (
	"archers"
	"context"
	"flag"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/sirupsen/logrus"
	"time"
)

var (
	archersNum = flag.Int("archers", 1, "amount of archers in the line")
	timeout    = flag.Int("timeout", 0, "timout in seconds before start messaging")
)

func main() {
	flag.Parse()
	ctx := context.Background()

	if archersNum != nil && *archersNum < 1 {
		logrus.Fatal("archers should be more than 0")
	}
	var node *peer.AddrInfo

	start := time.Now()
	firstArcher, err := archers.InitArchers(ctx, start, node, *archersNum)
	if err != nil {
		panic(err)
	}

	/* timeout */
	time.Sleep(time.Second * time.Duration(*timeout))
	/* start messaging */
	if err = firstArcher.Start(ctx); err != nil {
		panic(err)
	}
	time.Sleep(time.Duration(*archersNum) * 2 * time.Second)
}
