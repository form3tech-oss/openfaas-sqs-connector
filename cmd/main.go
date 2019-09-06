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

package main

import (
	"flag"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/openfaas-incubator/connector-sdk/types"
	log "github.com/sirupsen/logrus"

	"github.com/form3tech-oss/openfaas-sqs-connector/internal/processors"
)

func main() {
	// Parse command-line flags.
	logLevel := flag.String("log-level", "info", "the log level to use")
	maxNumberOfMessages := flag.Int64("max-number-of-messages", 1, "the maximum number of messages to return from the aws sqs queue per iteration")
	maxWaitTime := flag.Int64("max-wait-time", 1, "the maximum amount of time (in seconds) to wait for messages to be returned from the aws sqs queue per iteration")
	openfaasGatewayURL := flag.String("openfaas-gateway-url", "http://gateway.openfaas.svc:8080", "the url at which the openfaas gateway can be reached")
	queueURL := flag.String("queue-url", "", "the name of the aws sqs queue to pop messages from")
	region := flag.String("region", "", "the aws region to which the aws sqs queue belongs")
	topicRefreshInterval := flag.Int64("topic-refresh-interval", 15, "the interval (in seconds) at which to refresh the topic map")
	visibilityTimeout := flag.Int64("visibility-timeout", 30, "the amount of time (in seconds) during which received messages are unavailable to other consumers")
	flag.Parse()

	// Log at the requested level.
	if v, err := log.ParseLevel(*logLevel); err != nil {
		log.Fatalf("Failed to parse log level: %v", err)
	} else {
		log.SetLevel(v)
	}

	// Make sure that all required flags have been provided.
	if *region == "" {
		log.Fatal("--aws-region must be provided")
	}
	if *queueURL == "" {
		log.Fatal("--aws-sqs-queue-url must be provided")
	}
	if *openfaasGatewayURL == "" {
		log.Fatal("--gateway-url must be provided")
	}

	// Initialize the controller.
	controller := types.NewController(types.GetCredentials(), &types.ControllerConfig{
		GatewayURL:        *openfaasGatewayURL,
		PrintResponse:     log.IsLevelEnabled(log.DebugLevel),
		PrintResponseBody: log.IsLevelEnabled(log.DebugLevel),
		RebuildInterval:   time.Duration(*topicRefreshInterval) * time.Second,
	})
	controller.BeginMapBuilder()

	// Initialize the AWS SQS client.
	awsSession, err := session.NewSession(&aws.Config{
		CredentialsChainVerboseErrors: aws.Bool(log.IsLevelEnabled(log.DebugLevel)),
		Region:                        region,
	})
	if err != nil {
		log.Fatalf("Failed to initialize AWS session: %v", err)
	}
	awsSQSClient := sqs.New(awsSession)

	// Initialize the response processor.
	controller.Subscribe(processors.NewResponseProcessor(awsSQSClient, *queueURL))

	// Initialize the message processor and start processing messages off the AWS SQS queue.
	processors.NewMessageProcessor(awsSQSClient, *queueURL, *maxNumberOfMessages, *maxWaitTime, *visibilityTimeout, controller).Run()
}
