package consumer

import (
	"context"
	"fmt"
	"sync"

	"firefly.ai/word-counter/internal"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

const (
	awsMessageTimeoutBatch      = int32(10)
	awsWaitTimeSecondsBatch     = 10
	awsMaxNumberOfMessagesBatch = 10
)

var (
	deadLetterQueueName            string = "urls-dlq"
	deadLetterQueueMaxReceiveCount string = "3"
)

type Consumer struct {
	svc      *sqs.SQS
	queueURL string
}

func NewConsumer() *Consumer {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String("eu-central-1"),
		Endpoint: aws.String("http://localhost:4566"),
	}))

	svc := sqs.New(sess)

	return &Consumer{
		svc:      svc,
		queueURL: "http://localhost:4566/000000000000/url-queue",
	}
}

func (c *Consumer) ReadMessage(ctx context.Context, wg *sync.WaitGroup, sharedCounter *internal.WordCounter) error {
	// consume message annd delete it
	for {
		receiveMessageOutput, err := c.svc.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(c.queueURL),
			MaxNumberOfMessages: aws.Int64(awsMaxNumberOfMessagesBatch),
		})

		if err != nil {
			fmt.Println("Error:", err)
			return err
		}

		if len(receiveMessageOutput.Messages) == 0 {
			continue
		}

		// Process URL and get the word count
		url := *receiveMessageOutput.Messages[0].Body
		localCount, err := sharedCounter.CountWordsFromURL(ctx, url)
		if err != nil {
			fmt.Println("Error counting words:", err)
			continue
		}

		// Lock and update the shared counter
		sharedCounter.Sync.Lock()
		for word, count := range localCount {
			sharedCounter.Counter[word] += count
		}
		sharedCounter.Sync.Unlock()

		// Delete the message from the queue
		_, err = c.svc.DeleteMessage(&sqs.DeleteMessageInput{
			QueueUrl:      aws.String(c.queueURL),
			ReceiptHandle: receiveMessageOutput.Messages[0].ReceiptHandle,
		})

		if err != nil {
			fmt.Println("Delete Error", err)
			return err
		}

	}

}
