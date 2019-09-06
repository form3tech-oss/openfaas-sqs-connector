// Copyright 2019 Form3 Financial Cloud
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package processors

import (
	"sync"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/openfaas-incubator/connector-sdk/types"
	log "github.com/sirupsen/logrus"

	"github.com/form3tech-oss/openfaas-sqs-connector/internal/pointers"
)

const (
	messageAttributeNameAll   = "All"
	messageAttributeNameTopic = "Topic"
)

// MessageProcessor reads and processes messages off of an AWS SQS queue.
type MessageProcessor struct {
	client                   *sqs.SQS
	controller               *types.Controller
	maxNumberOfMessages      int64
	maxWaitTimeSeconds       int64
	queueURL                 string
	visibilityTimeoutSeconds int64
}

// NewMessageProcessor creates a new instance of MessageProcessor.
func NewMessageProcessor(awsSQSClient *sqs.SQS, awsSQSQueueURL string, awsSQSQueueMaxNumberOfMessages int64, awsSQSQueueMaxWaitTimeSeconds int64, awsSQSQueueVisibilityTimeoutSeconds int64, controller *types.Controller) *MessageProcessor {
	return &MessageProcessor{
		client:                   awsSQSClient,
		controller:               controller,
		maxNumberOfMessages:      awsSQSQueueMaxNumberOfMessages,
		maxWaitTimeSeconds:       awsSQSQueueMaxWaitTimeSeconds,
		queueURL:                 awsSQSQueueURL,
		visibilityTimeoutSeconds: awsSQSQueueVisibilityTimeoutSeconds,
	}
}

// Run sits on a loop reading and processing messages off of the AWS SQS queue.
func (p *MessageProcessor) Run() {
	for {
		r, err := p.client.ReceiveMessage(&sqs.ReceiveMessageInput{
			MaxNumberOfMessages: aws.Int64(p.maxNumberOfMessages),
			MessageAttributeNames: []*string{
				aws.String(messageAttributeNameAll),
			},
			QueueUrl:          aws.String(p.queueURL),
			VisibilityTimeout: aws.Int64(p.visibilityTimeoutSeconds),
			WaitTimeSeconds:   aws.Int64(p.maxWaitTimeSeconds),
		})
		if err != nil {
			log.Errorf("Failed to receive message: %v", err)
			continue
		}
		if len(r.Messages) < 1 {
			continue
		}

		var wg sync.WaitGroup
		wg.Add(len(r.Messages))
		for _, message := range r.Messages {
			go func(message *sqs.Message) {
				defer wg.Done()
				log.Tracef("Processing message with id %q", *message.MessageId)

				var (
					body  string
					topic string
				)

				// Retrieve the message's body (if any).
				if message.Body == nil {
					body = ""
				} else {
					body = *message.Body
				}

				// Retrieve the message's topic (if any).
				if v, ok := message.MessageAttributes[messageAttributeNameTopic]; !ok || v == nil || *v.StringValue == "" {
					topic = p.queueURL
				} else {
					topic = *v.StringValue
				}

				// Invoke the function(s) associated with the topic.
				p.controller.InvokeWithContext(buildMessageContext(message), topic, pointers.NewBytes([]byte(body)))
			}(message)
		}
		wg.Wait()
	}
}
