package fuzzer

import (
	"bufio"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/mcastellin/turbo-intruder/pkg/domain"
)

const defaultBatchSize = 100
const clientPoolSize = 25
const defaultDeadlineSeconds = 60

// PooledPipelinedClient is an HTTP client capable of
// performing a high volume of requests against the same host
// using and HTTP/1.1 feature called HTTP pipelining
type PooledPipelinedClient struct {
	INC        chan domain.Wrapper
	OUTC       chan domain.FuzzResponse
	doneCh     chan struct{}
	clientPool []pipelinedClient
}

// NewPooledPipelinedClient initialises a new instance of the client pool
func NewPooledPipelinedClient() *PooledPipelinedClient {
	clientPool := make([]pipelinedClient, clientPoolSize)
	inCh := make(chan domain.Wrapper, defaultBatchSize)
	outCh := make(chan domain.FuzzResponse, defaultBatchSize)

	for i := 0; i < len(clientPool); i++ {
		clientPool[i] = pipelinedClient{
			INC:       inCh,
			OUTC:      outCh,
			batch:     make([]*domain.Wrapper, defaultBatchSize),
			taintConn: true,
		}
	}
	return &PooledPipelinedClient{
		INC:        inCh,
		OUTC:       outCh,
		doneCh:     make(chan struct{}),
		clientPool: clientPool,
	}
}

// Start message processing for all clients in the pool
func (pc *PooledPipelinedClient) Start() {
	for _, client := range pc.clientPool {
		if err := client.Start(); err != nil {
			panic(err)
		}
	}
}

// Close incoming channel and wait for clients to
// gracefully terminate
func (pc *PooledPipelinedClient) Close() {
	close(pc.INC)
	for i := 0; i < len(pc.clientPool); i++ {
		<-pc.doneCh
	}
	close(pc.doneCh)
	close(pc.OUTC)
}

type pipelinedClient struct {
	INC  <-chan domain.Wrapper
	OUTC chan<- domain.FuzzResponse

	started  bool
	batch    []*domain.Wrapper
	batchPtr int

	conn      net.Conn
	taintConn bool
	writer    *bufio.Writer
	reader    *bufio.Reader
}

func (c *pipelinedClient) Start() error {
	processFn := func() {
		// at this point the net connection is not initialised,
		// though we defer connection close at routine exit
		defer func() {
			if !c.taintConn {
				c.conn.Close()
			}
		}()

		c.batchPtr = 0
		for {
			select {
			case w, more := <-c.INC:
				if !more {
					c.flushRequests()
					return
				}
				c.batch[c.batchPtr] = &w
				c.batchPtr += 1
				if c.batchPtr >= len(c.batch) {
					c.flushRequests()
				}
			}
		}
	}

	if c.started {
		// only one routine can run at any time for pipelinedClient.
		// this is important as we need to maintain the sequence of read/write
		// operations for every requests batch sharing the same tcp connection
		return fmt.Errorf("pipelinedClient already started")
	}

	go processFn()
	c.started = true
	return nil
}

// flushRequests is the HTTP pipelining logic. Once a batch of incoming requests
// is complete or the ingress channel is closed, the entire batch is processed
// at once.
//
// Given the batched nature of this operation, not all batches will complete
// processing in a single pass. If the TCP connection is closed while reading
// responses, incomplete requests will be re-processed from scratch using a
// fresh connection.
func (c *pipelinedClient) flushRequests() {
	processBatch := func(startPtr int) (int, error) {
		if c.taintConn {
			if startPtr >= c.batchPtr {
				return 0, nil
			}
			c.initConn(c.batch[startPtr].Host)
		}

		for i := startPtr; i < c.batchPtr; i++ {
			if _, err := c.writer.WriteString(c.batch[i].Request); err != nil {
				return 0, err
			}
		}
		c.writer.Flush()

		numProcessed := startPtr
		for i := startPtr; i < c.batchPtr; i++ {
			resp, err := http.ReadResponse(c.reader, nil)
			if err != nil {
				return numProcessed, err
			}
			body := make([]byte, resp.ContentLength)
			n, _ := resp.Body.Read(body)
			resp.Body.Close()

			fr := domain.FuzzResponse{
				Req:        *c.batch[i],
				Body:       string(body[:n]),
				Status:     resp.Status,
				StatusCode: resp.StatusCode,
			}
			c.OUTC <- fr
			numProcessed += 1

			ka := resp.Header["Connection"]
			if len(ka) == 0 || ka[0] != "keep-alive" {
				// TCP connection has been closed.
				// We expect all further reads from the conn to fail.
				c.taintConn = true
				return numProcessed, nil
			}

		}
		return numProcessed, nil
	}

	startPtr := 0
	for {
		processed, err := processBatch(startPtr)
		if err != nil {
			panic(err)
		}
		if startPtr+processed == c.batchPtr {
			c.batchPtr = 0
			return
		}
		// could not process all but no error returned
		// probably means the connection has been closed.
		// creating a new connection and continue processing.
		startPtr += processed
	}
}

func (c *pipelinedClient) initConn(host string) {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		panic(err)
	}
	conn.SetDeadline(time.Now().Add(defaultDeadlineSeconds * time.Second))
	c.conn = conn
	c.writer = bufio.NewWriter(conn)
	c.reader = bufio.NewReader(conn)
}
