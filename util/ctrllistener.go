package util

import (
	"bufio"
	"fmt"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
)

type CtrlListener struct{
	listener net.Listener
	callbacks map[string]func(string) error
}

func NewCtrlListener(root, id string) (cl *CtrlListener, err error) {
	cl = &CtrlListener{callbacks: make(map[string]func(string) error)}
	address := filepath.Join(root, fmt.Sprintf("%s.%d.sock", id, os.Getpid()))
	unixAddress, err := net.ResolveUnixAddr("unix", address)
	if err != nil {
		return nil, errors.Wrap(err, "error resolving unix address")
	}
	cl.listener, err = net.ListenUnix("unix", unixAddress)
	return cl, nil
}

func (self *CtrlListener) AddCallback(keyword string, f func(string) error) {
	self.callbacks[keyword] = f
}

func (self *CtrlListener) Start() {
	go self.run()
}

func (self *CtrlListener) run() {
	logrus.Infof("[%s] started", self.listener.Addr())
	defer logrus.Infof("[%s] exited", self.listener.Addr())

	for {
		conn, err := self.listener.Accept()
		if err == nil {
			go self.handle(conn)
		} else if err == io.EOF {
			return
		} else if err != nil {
			logrus.Errorf("error accepting ctrl connection (%v)", err)
		}
	}
}

func (self *CtrlListener) handle(conn net.Conn) {
	logrus.Infof("new connection for [%s]", conn.LocalAddr())
	defer logrus.Infof("ended connection for [%s]", conn.LocalAddr())

	r := bufio.NewReader(conn)
	for {
		line, err := r.ReadString('\n')
		if err == io.EOF {
			return
		} else if err != nil {
			logrus.Errorf("error reading (%v)", err)
			return
		}

		line = strings.TrimSpace(line)
		tokens := strings.Split(line, " ")
		if len(tokens) > 0 {
			f, found := self.callbacks[tokens[0]]
			if found {
				fErr := f(line)
				if fErr == nil {
					_, err := conn.Write([]byte("ok\n"))
					if err != nil {
						logrus.Errorf("error responding (%v)", err)
					}
				} else {
					logrus.Errorf("error executing callback (%v)", fErr)
					_, err := conn.Write([]byte(fmt.Sprintf("error (%s)\n", fErr)))
					if err != nil {
						logrus.Errorf("error responding (%v)", err)
					}
				}
			} else {
				logrus.Errorf("no callback for [%s]", line)
				_, err := conn.Write([]byte("syntax error?\n"))
				if err != nil {
					logrus.Errorf("error responding (%v)", err)
				}
			}
		} else {
			logrus.Errorf("no tokens")
			_, err := conn.Write([]byte("syntax error?\n"))
			if err != nil {
				logrus.Errorf("error responding (%v)", err)
			}
		}
	}
}