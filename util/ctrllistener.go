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
	"sync"
)

var ctrlListeners map[string]*CtrlListener
var ctrlMutex sync.Mutex

type CtrlListener struct {
	listener  net.Listener
	callbacks map[string][]func(string) error
	running   bool
}

func GetCtrlListener(root, id string) (cl *CtrlListener, err error) {
	ctrlMutex.Lock()
	defer ctrlMutex.Unlock()

	cl, found := ctrlListeners[root+id]
	if found {
		return cl, nil
	}

	cl = &CtrlListener{callbacks: make(map[string][]func(string) error)}
	address := filepath.Join(root, fmt.Sprintf("%s.%d.sock", id, os.Getpid()))
	unixAddress, err := net.ResolveUnixAddr("unix", address)
	if err != nil {
		return nil, errors.Wrap(err, "error resolving unix address")
	}
	cl.listener, err = net.ListenUnix("unix", unixAddress)
	if err != nil {
		return nil, errors.Wrap(err, "error listening")
	}
	return cl, nil
}

func (self *CtrlListener) AddCallback(keyword string, f func(string) error) {
	self.callbacks[keyword] = append(self.callbacks[keyword], f)
}

func (self *CtrlListener) Start() {
	ctrlMutex.Lock()
	defer ctrlMutex.Unlock()

	if !self.running {
		go self.run()
	}
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
			fs, found := self.callbacks[tokens[0]]
			if found {
				var fErr error
			fsLoop:
				for _, f := range fs {
					fErr = f(line)
					if fErr != nil {
						break fsLoop
					}
				}
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
