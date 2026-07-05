package worker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"

	"github.com/Sakayori-Iroha-168/Software_Teamwork/services/document/internal/service"
	"github.com/hibiken/asynq"
)

type Client struct {
	client *asynq.Client
}

type RedisConfig struct {
	Addr       string
	Username   string
	Password   string
	DB         int
	TLSEnabled bool
}

func NewClient(redis RedisConfig) *Client {
	return &Client{
		client: asynq.NewClient(redisClientOpt(redis)),
	}
}

func redisClientOpt(redis RedisConfig) asynq.RedisClientOpt {
	opt := asynq.RedisClientOpt{
		Addr:     redis.Addr,
		Username: redis.Username,
		Password: redis.Password,
		DB:       redis.DB,
	}
	if redis.TLSEnabled {
		opt.TLSConfig = &tls.Config{MinVersion: tls.VersionTLS12}
	}
	return opt
}

func (c *Client) Close() error {
	return c.client.Close()
}

// EnqueueReportJob implements service.TaskEnqueuer.
func (c *Client) EnqueueReportJob(ctx context.Context, jobType service.JobType, jobID, attemptID, requestID, userID string) (string, error) {
	taskType, err := TaskTypeForJobType(jobType)
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(ReportJobPayload{
		RequestID: requestID,
		JobType:   string(jobType),
		JobID:     jobID,
		AttemptID: attemptID,
		UserID:    userID,
	})
	if err != nil {
		return "", fmt.Errorf("marshal report job payload: %w", err)
	}
	task := asynq.NewTask(taskType, data, asynq.Queue("document"))
	info, err := c.client.EnqueueContext(ctx, task)
	if err != nil {
		return "", fmt.Errorf("enqueue report job: %w", err)
	}
	return info.ID, nil
}
