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
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/openfaas-incubator/connector-sdk/types"
)

// ResponseProcessor processes responses to functions invocations made while processing messages off of an AWS SQS queue.
type ResponseProcessor struct {
	client   *sqs.SQS
	queueURL string
}

// NewResponseProcessor creates a new instance of ResponseProcessor.
func NewResponseProcessor(awsSQSClient *sqs.SQS, awsSQSQueueURL string) *ResponseProcessor {
	return &ResponseProcessor{
		client:   awsSQSClient,
		queueURL: awsSQSQueueURL,
	}
}

// Response is invoked whenever a response to a given function invocation is received.
func (r *ResponseProcessor) Response(res types.InvokerResponse) {
	// Handle processing of the response in a separate goroutine.
	// https://github.com/openfaas-incubator/connector-sdk/blob/0.4.2/types/response_subscriber.go#L5-L7
	go func() {
		logger, _, receiptHandle := unpackMessageContext(res.Context)

		if res.Error != nil || res.Status >= http.StatusMultipleChoices {
			// Log the error.
			logger.Warnf("Failed to process message: %v", res.Error)
			// Change the message's visibility so it can be picked up immediately by another consumer.
			_, err := r.client.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
				QueueUrl:          aws.String(r.queueURL),
				ReceiptHandle:     aws.String(receiptHandle),
				VisibilityTimeout: aws.Int64(0),
			})
			if err != nil {
				logger.Errorf("Failed to change message visibility: %v", err)
			} else {
				logger.Trace("Message visibility successfully changed")
			}
		} else {
			// Log the fact that the message was successfully processed.
			logger.Trace("Message successfully processed")
			// Delete the message from the queue.
			_, err := r.client.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      aws.String(r.queueURL),
				ReceiptHandle: aws.String(receiptHandle),
			})
			if err != nil {
				logger.Errorf("Failed to delete message: %v", err)
			} else {
				logger.Trace("Message successfully deleted from the queue")
			}
		}
	}()
}
