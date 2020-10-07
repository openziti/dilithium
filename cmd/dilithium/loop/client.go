package loop

import (
	"github.com/michaelquigley/dilithium/protocol/loop"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"net"
)

func init() {
	loopCmd.AddCommand(loopClientCmd)
}

var loopClientCmd = &cobra.Command{
	Use:   "client <serverAddress>",
	Short: "Start loop client",
	Args:  cobra.ExactArgs(1),
	Run:   loopClient,
}

func loopClient(_ *cobra.Command, args []string) {
	var ds *loop.DataSet
	if startSender {
		var err error
		ds, err = loop.NewDataSet(2 + 64 + size)
		if err != nil {
			logrus.Fatalf("error creating dataset (%v)", err)
		}
	}

	serverAddress, err := net.ResolveTCPAddr("tcp", args[0])
	if err != nil {
		logrus.Fatalf("error parsing server address (%v)", err)
	}

	conn, err := net.DialTCP("tcp", nil, serverAddress)
	if err != nil {
		logrus.Fatalf("error dialing server (%v)", err)
	}

	var rx *loop.Receiver
	if startReceiver {
		rx = loop.NewReceiver(conn)
		go rx.Run(startHasher)
	}

	var tx *loop.Sender
	if startSender {
		tx = loop.NewSender(ds, conn, count)
		go tx.Run()
	}

	if rx != nil {
		<-rx.Done
	}
	if tx != nil {
		<-tx.Done
	}
}
