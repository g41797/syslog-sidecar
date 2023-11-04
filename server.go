package syslogsidecar

import (
	"sync/atomic"

	"github.com/g41797/go-syslog"
	"github.com/g41797/go-syslog/format"
	"github.com/g41797/kissngoqueue"
	"github.com/g41797/sputnik"
)

type SyslogConfiguration struct {
	// The Syslog Severity level ranges between 0 to 7.
	// Each number points to the relevance of the action reported.
	// From a debugging message (7) to a completely unusable system (0):
	//
	//	0		Emergency: system is unusable
	//	1		Alert: action must be taken immediately
	//	2		Critical: critical conditions
	//	3		Error: error conditions
	//	4		Warning: warning conditions
	//	5		Notice: normal but significant condition
	//	6		Informational: informational messages
	//	7		Debug: debug-level messages
	//
	// Log with severity above value from configuration will be discarded
	// Examples:
	// -1 - all logs will be discarded
	// 5  - logs with severities 6(Informational) and 7(Debug) will be discarded
	// 7  - all logs will be processed
	SEVERITYLEVEL int

	// IPv4 address of TCP listener.
	// For empty string - don't use TCP
	// e.g "0.0.0.0:5141" - listen on all adapters, port 5141
	// "127.0.0.1:5141" - listen on loopback "adapter"
	ADDRTCP string

	// IPv4 address of UDP receiver.
	// For empty string - don't use UDP
	// Usually "0.0.0.0:5141" - receive from all adapters, port 5141
	// "127.0.0.1:5141" - receive from loopback "adapter"
	ADDRUDP string

	// Unix domain socket name - actually file path.
	// For empty string - don't use UDS
	// Regarding limitations see https://man7.org/linux/man-pages/man7/unix.7.html
	UDSPATH string

	// TLS section: Listening on non empty ADDRTCPTLS will start only
	// for valid tls configuration (created using last 3 parameters)
	ADDRTCPTLS       string
	CLIENT_CERT_PATH string
	CLIENT_KEY_PATH  string
	ROOT_CA_PATH     string
}

type syslogs []*syslog.Server

type server struct {
	config SyslogConfiguration
	bc     atomic.Pointer[sputnik.BlockCommunicator]
	logs   syslogs
	q      *kissngoqueue.Queue[format.LogParts]
}

func newServer(conf SyslogConfiguration) *server {
	srv := new(server)
	srv.config = conf
	srv.bc = atomic.Pointer[sputnik.BlockCommunicator]{}
	srv.q = kissngoqueue.NewQueue[format.LogParts]()
	srv.logs = make(syslogs, 0)
	return srv
}

func (s *server) initServer() error {

	if err := s.newsyslogdTCP(); err != nil {
		return err
	}

	if err := s.newsyslogdTCPTLS(); err != nil {
		return err
	}

	if err := s.newsyslogdUDS(); err != nil {
		return err
	}

	ls, err := s.newsyslogdUDP()
	if err != nil {
		return err
	}

	if ls != nil {

		s.logs = append(s.logs, ls)

		if ls.IsUDPReusable() {
			for i := 1; i < 8; i++ {
				ls, _ := s.newsyslogdUDP()
				s.logs = append(s.logs, ls)
			}
		}
	}

	go s.processLogParts()

	return nil
}

func (s *server) newsyslogd() *syslog.Server {
	result := syslog.NewServer()
	result.SetFormat(syslog.Automatic)
	result.SetHandler(s)
	return result
}

func (s *server) newsyslogdTCP() error {

	if len(s.config.ADDRTCP) == 0 {
		return nil
	}

	ls := s.newsyslogd()

	if err := ls.ListenTCP(s.config.ADDRTCP); err != nil {
		return err
	}

	s.logs = append(s.logs, ls)

	return nil
}

func (s *server) newsyslogdTCPTLS() error {

	if len(s.config.ADDRTCPTLS) == 0 {
		return nil
	}

	t, err := prepareTLS(s.config.CLIENT_CERT_PATH, s.config.CLIENT_KEY_PATH, s.config.ROOT_CA_PATH)

	if err != nil {
		return err
	}

	if t == nil {
		return nil
	}

	ls := s.newsyslogd()

	if err = ls.ListenTCPTLS(s.config.ADDRTCPTLS, t); err != nil {
		return err
	}

	s.logs = append(s.logs, ls)

	return nil
}

func (s *server) newsyslogdUDS() error {

	if len(s.config.UDSPATH) == 0 {
		return nil
	}

	ls := s.newsyslogd()

	if err := ls.ListenUnixgram(s.config.UDSPATH); err != nil {
		return err
	}

	s.logs = append(s.logs, ls)

	return nil
}

func (s *server) newsyslogdUDP() (*syslog.Server, error) {

	if len(s.config.ADDRUDP) == 0 {
		return nil, nil
	}

	ls := s.newsyslogd()

	if err := ls.ListenUDP(s.config.ADDRUDP); err != nil {
		return nil, err
	}

	return ls, nil
}

func (s *server) start() error {
	return s.logs.Boot()
}

func (s *server) stop() error {
	s.q.CancelMT()
	return s.logs.Kill()
}

func (s *server) setupHandling(bc sputnik.BlockCommunicator) {
	s.bc.Store(&bc)
}

// Process received and parsed syslog messages - called by go-syslog
func (s *server) Handle(logParts format.LogParts, msgLen int64, err error) {
	if s.bc.Load() == nil {
		return
	}

	if (err == nil) && (!s.forHandle(logParts)) {
		return
	}

	s.q.PutMT(logParts)
}

func (s *server) processLogParts() {
	for {
		lp, ok := s.q.Get()
		if !ok {
			break
		}
		(*s.bc.Load()).Send(toMsg(lp))
	}
	return
}

func (s *server) forHandle(logParts format.LogParts) bool {
	if s.config.SEVERITYLEVEL == -1 {
		return false
	}

	if logParts == nil {
		return false
	}

	if len(logParts) == 0 {
		return false
	}

	severity, exists := logParts[severityKey]

	if !exists {
		return true
	}

	sevvalue, _ := severity.(int)

	return sevvalue <= s.config.SEVERITYLEVEL
}

func (logs syslogs) Boot() error {
	if len(logs) == 0 {
		return nil
	}

	var err error

	for _, l := range logs {
		if err = l.Boot(); err != nil {
			return err
		}
	}

	return nil
}

func (logs syslogs) Kill() error {
	if len(logs) == 0 {
		return nil
	}

	for _, l := range logs {
		l.Kill()
	}
	return nil
}
