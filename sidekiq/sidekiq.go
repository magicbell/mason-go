// Package sidekiq provides methods interact with sidekiq, used for scheduling jobs
// in the legacy Rails app.
package sidekiq

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"time"

	"go.uber.org/zap"
	redis "gopkg.in/redis.v5"
)

type Client struct {
	rdb *redis.Client
	log *zap.SugaredLogger
}

type Job struct {
	Queue      string        `json:"queue"`
	Class      string        `json:"class"`
	Args       []interface{} `json:"args"`
	JID        string        `json:"jid"`
	Retry      bool          `json:"retry"`
	CreatedAt  float64       `json:"created_at"`
	EnqueuedAt float64       `json:"enqueued_at"`
}

func (j Job) MarshalBinary() ([]byte, error) {
	return json.Marshal(j)
}

func New(rdb *redis.Client, log *zap.SugaredLogger) *Client {
	return &Client{
		rdb: rdb,
		log: log,
	}
}

func (c *Client) Enqueue(job string, queue string, args ...interface{}) (string, error) {

	id := make([]byte, 12)
	io.ReadFull(rand.Reader, id)
	jid := hex.EncodeToString(id)

	ts := float64(time.Now().Unix())

	j := Job{
		Queue:      queue,
		Class:      job,
		Args:       args,
		JID:        jid,
		Retry:      true,
		CreatedAt:  ts,
		EnqueuedAt: ts,
	}

	err := c.rdb.SAdd("queues", queue).Err()
	if err != nil {
		return jid, err
	}

	err = c.rdb.LPush("queue:"+queue, j).Err()
	if err != nil {
		return jid, err
	}

	c.log.Infow("enqueued to sidekiq", "jid", jid, "enqueued_at", ts, "queue", queue, "job", job, "args", args)

	return jid, nil
}
