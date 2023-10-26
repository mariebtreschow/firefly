package producer

import (
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

type Producer struct {
	svc      *sqs.SQS
	queueURL string
}

func NewProducer() *Producer {
	sess := session.Must(session.NewSession(&aws.Config{
		Region:   aws.String("eu-central-1"),
		Endpoint: aws.String("http://localhost:4566"),
	}))

	svc := sqs.New(sess)

	return &Producer{
		svc:      svc,
		queueURL: "http://localhost:4566/000000000000/url-queue",
	}
}

func (p *Producer) PublishMessage(urls []string) error {
	// check if queue exists and create it if it does not
	_, err := p.svc.GetQueueUrl(&sqs.GetQueueUrlInput{
		QueueName: aws.String("url-queue"),
	})
	if err != nil {
		// create queue
		_, err := p.svc.CreateQueue(&sqs.CreateQueueInput{
			QueueName: aws.String("url-queue"),
		})
		if err != nil {
			log.Println("Error:", err)
			return err
		}

	}
	for _, url := range urls {
		_, err := p.svc.SendMessage(&sqs.SendMessageInput{
			MessageBody: aws.String(url),
			QueueUrl:    aws.String(p.queueURL),
		})
		if err != nil {
			log.Println("Error:", err)
			return err
		}
	}
	fmt.Println("Successfully sent urls with length", len(urls))
	return nil
}
