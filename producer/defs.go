package producer

import (
	"net"
	"os/exec"
	"sync"
	"time"

	"github.com/couchbase/eventing/suptree"
	cbbucket "github.com/couchbase/go-couchbase"
	"github.com/couchbase/indexing/secondary/dcp"
	mcd "github.com/couchbase/indexing/secondary/dcp/transport"
	cb "github.com/couchbase/indexing/secondary/dcp/transport/client"
)

const (
	// Folder containing all eventing app definition and configs
	APPS_FOLDER = "./apps/"

	EVENTING_ADMIN_SERVICE = "eventingAdminPort"
	DATA_SERVICE           = "kv"
	MGMT_SERVICE           = "mgmt"

	NUM_VBUCKETS = 1024

	// DCP consumer related configs
	DCP_GEN_CHAN_SIZE   = 10000
	DCP_DATA_CHAN_SIZE  = 10000
	DCP_NUM_CONNECTIONS = 4

	// Last processed seq # checkpoint interval, in seconds
	CHECKPOINT_INTERVAL = 1

	// Interval for retrying failed bucket operations using go-couchbase, in milliseconds
	BUCKET_OP_RETRY_INTERVAL = 100

	// Interval for spawning another routine to keep an eye on cluster state change, in milliseconds
	WATCH_CLUSTER_CHANGE_INTERVAL = 100

	// Interval for polling in order to take ownership of desired vbucket, in seconds
	VB_TAKEOVER_POLL_INTERVAL = 1
)

const (
	DCP_STREAM_RUNNING = "running"
	DCP_STREAM_STOPPED = "stopped"
)

type vbFlogEntry struct {
	seqNo          uint64
	streamReqRetry bool
	statusCode     mcd.Status
	vb             uint16
	flog           *cb.FailoverLog
}

type Consumer struct {
	app  *appConfig
	conn net.Conn

	dcpFeed  *couchbase.DcpFeed
	cbBucket *couchbase.Bucket
	vbnos    []uint16

	// Map that needed to short circuits failover log to dcp stream request routine
	vbFlogChan chan *vbFlogEntry

	// host:port handle for current eventing node
	hostPortAddr string

	workerName string
	producer   *Producer

	metadataBucketHandle *cbbucket.Bucket

	// OS pid of c++ v8 worker
	osPid int

	// C++ v8 worker cmd handle, would be required to killing worker that are no more needed
	cmd *exec.Cmd

	// Populated when C++ v8 worker is spawned
	// correctly and downstream tcp socket is available
	// for sending messages. Unbuffered channel.
	signalConnectedCh chan bool

	// Populated when downstream tcp socket mapping to
	// C++ v8 worker is down. Buffered channel to avoid deadlock
	stopConsumerCh chan bool

	// Chan to stop background checkpoint routine, keeping track
	// of last seq # processed
	stopCheckpointingCh chan bool

	// Chan to stop background vb takeover polling routine
	stopVbTakeoverCh chan bool

	gracefulShutdownChan chan bool

	tcpPort string

	// Tracks DCP Opcodes processed per consumer
	dcpMessagesProcessed map[mcd.CommandCode]int

	// Tracks V8 Opcodes processed per consumer
	v8WorkerMessagesProcessed map[string]int

	sync.RWMutex
	vbProcessingStats vbStats

	statsTicker      *time.Ticker
	checkpointTicker *time.Ticker
	vbTakeoverTicker *time.Ticker
}

type vbStats map[uint16]*vbStat

type vbStat struct {
	stats map[string]interface{}
	sync.RWMutex
}

type Producer struct {
	AppName string

	app              *appConfig
	auth             string
	bucket           string
	KvPort           string
	kvHostPort       []string
	NsServerPort     string
	nsServerHostPort string
	tcpPort          string
	stopProducerCh   chan bool
	workerCount      int

	// stats gathered from ClusterInfo
	localAddress      string
	eventingNodeAddrs []string
	kvNodeAddrs       []string
	nsServerNodeAddrs []string

	// Feedback channel to notify change in cluster state
	clusterStateChange chan bool

	// List of running consumers, will be needed if we want to gracefully shut them down
	runningConsumers           []*Consumer
	consumerSupervisorTokenMap map[*Consumer]suptree.ServiceToken

	// vbucket to eventing node assignment
	vbEventingNodeAssignMap map[uint16]string

	// copy of KV vbmap, needed while opening up dcp feed
	kvVbMap map[uint16]string

	// time.Ticker duration for dumping consumer stats
	statsTickDuration time.Duration

	// Map keeping track of start and end vbucket
	// for each worker
	workerVbucketMap map[int][]uint16

	// Supervisor of workers responsible for
	// pipelining messages to V8
	workerSupervisor *suptree.Supervisor

	sync.RWMutex
}

type appConfig struct {
	AppName string      `json:"appname"`
	AppCode string      `json:"appcode"`
	Depcfg  interface{} `json:"depcfg"`
	ID      int         `json:"id"`
}

type vbucketKVBlob struct {
	CurrentVBOwner     string `json:"current_vb_owner"`
	DCPStreamStatus    string `json:"dcp_stream_status"`
	LastCheckpointTime string `json:"last_checkpoint_time"`
	LastSeqNoProcessed uint64 `json:"last_processed_seq_no"`
	NewVBOwner         string `json:"new_vb_owner"`
	VBId               uint16 `json:"vb_id"`
	VBuuid             uint64 `json:"vb_uuid"`
}
