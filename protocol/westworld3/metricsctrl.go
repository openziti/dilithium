package westworld3

import (
	"bufio"
	"fmt"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
)

type metricsInstrumentController struct{
	l net.Listener
}

func NewMetricsInstrumentController(root string, mi *metricsInstrument) (mic *metricsInstrumentController, err error) {
	mic = &metricsInstrumentController{}
	laddr, lerr := net.ResolveUnixAddr("unix", socketPath(root))
	if lerr != nil {
		err = lerr
		return
	}
	mic.l, err = net.ListenUnix("unix", laddr)
	go mic.run()
	return mic, err
}

func (self *metricsInstrumentController) run() {
	logrus.Infof("[%s] started", self.l.Addr())
	defer logrus.Infof("[%s] exited", self.l.Addr())

	for {
		conn, err := self.l.Accept()
		if err == nil {
			go self.handle(conn)
		} else if err == io.EOF {
			return
		} else  if err != nil {
			logrus.Errorf("error accepting metrics ctrl connection (%v)", err)
		}
	}
}

func (self *metricsInstrumentController) handle(conn net.Conn) {
	logrus.Infof("handling connection [%s]", conn.RemoteAddr())
	defer logrus.Infof("ending connection [%s]", conn.RemoteAddr())

	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			return
		} else if err != nil {
			logrus.Errorf("error reading (%v)", err)
			return
		}

		logrus.Infof("line = [%s]", line)
		_, _ = conn.Write([]byte("ok\n"))
	}
}

func socketPath(root string) string {
	return fmt.Sprintf("%s/westworld3.%d.sock", root, os.Getpid())
}
