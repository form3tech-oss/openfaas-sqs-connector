# openfaas-sqs-connector

[![Build Status](https://travis-ci.com/form3tech-oss/openfaas-sqs-connector.svg?branch=master)](https://travis-ci.com/form3tech-oss/openfaas-sqs-connector)

An OpenFaaS connector for AWS SQS.

## Goals

* Allow [OpenFaaS](https://www.openfaas.com/) functions to be invoked as a result of messages being sent to an [AWS SQS](https://aws.amazon.com/sqs/) queue.
* Allow for multiplexing different [topics](https://docs.openfaas.com/reference/triggers/#event-connector-pattern) over a single AWS SQS queue.

## Usage

To start receiving messages from an SQS queue in your OpenFaas functions:
* Install an instance of openfaas-sqs-connector configured to receive from the queue.
* Add a `topic` annotation to each OpenFaaS function denoting which topic you'd like to receive. Alternatively you can set the topic to the queue url to receive all messages from that queue.
* Add messages to the SQS queue with a `Topic` message attribute to route to specific functions.

## Installing

### Helm (Experimental)

To install `openfaas-sqs-connector` using Helm, run

```shell
$ helm repo add openfaas-sqs-connector https://form3tech-oss.github.io/openfaas-sqs-connector
```

```shell
$ helm repo update
```

```shell
$ helm upgrade --install openfaas-sqs-connector openfaas-sqs-connector/openfaas-sqs-connector \
  --namespace openfaas \
  --set queueUrl=<queue-url> \
  --set region=<region>
```

Please check [`values.yaml`](https://github.com/form3tech-oss/openfaas-sqs-connector/blob/master/helm/openfaas-sqs-connector/values.yaml) for details on how to tweak the installation. 

### `kubectl`

To install `openfaas-sqs-connector` using `kubectl`, edit `./deploy/kubernetes/openfaas-sqs-connector-dep.yaml` as required in order to set appropriate values for each flag:

Flag&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;&nbsp; | Description | Default
---- | ----------- | -------
`--endpoint` | The AWS SQS endpoint to use. Useful when using [`elasticmq`](https://github.com/softwaremill/elasticmq) for local development. | `""`
`--log-level` | The log level to use. | `info`
`--max-number-of-messages` | The maximum number of messages to return from the AWS SQS queue per iteration. | `1`
`--max-wait-time` | The maximum amount of time (in seconds) to wait for messages to be returned from the AWS SQS queue per iteration. | `1`
`--openfaas-gateway-url` | The URL at which the OpenFaaS gateway can be reached. | `http://gateway.openfaas.svc:8080`
`--queue-url` | The name of the AWS SQS queue to pop messages from. | N/A
`--region` | The AWS region to which the AWS SQS queue belongs. | N/A
`--topic-refresh-interval` | The interval (in seconds) at which to rebuild the topic map. | `15`
`--visibility-timeout` | The amount of time (in seconds) during which received messages are unavailable to other consumers. | `30`

Then, run

```shell
$ kubectl create -f ./deploy/kubernetes/openfaas-sqs-connector-dep.yaml
```

### Permissions

You will need to make sure your Kubernetes worker nodes on which the SQS Connector will be deployed have a role with the following IAM permissions to interact with your SQS queues:

- `sqs:DeleteMessage`
- `sqs:ChangeMessageVisibility`
- `sqs:ReceiveMessage`

## Testing

### Kubernetes

To test `openfaas-sqs-connector`, start by deploying two functions:

```shell
$ faas store deploy figlet --annotation topic="figlet"
Deployed. 202 Accepted.
URL: https://openfaas.example.com/function/figlet
```

```
$ faas store deploy nslookup --annotation topic="nslookup"
Deployed. 202 Accepted.
URL: https://openfaas.example.com/function/nslookup
```

Now, send two messages to the AWS SQS queue from which `openfaas-sqs-connector` is consuming messages:

```shell
$ aws sqs send-message \
    --queue-url https://sqs.eu-west-1.amazonaws.com/<id>/openfaas-sqs-connector \
    --message-attributes '{"Topic":{"DataType":"String","StringValue":"figlet"}}' \
    --message-body "openfaas"
{
    "MD5OfMessageBody": "f6650c4b9f5c1c7c627a9d150da3461b",
    "MD5OfMessageAttributes": "dc49cdc863571fa19ce0f748008752ba",
    "MessageId": "daad6027-9503-4952-b1de-4b29314f2048"
}
```

```shell
$ aws sqs send-message \
    --queue-url https://sqs.eu-west-1.amazonaws.com/<id>/openfaas-sqs-connector \
    --message-attributes '{"Topic":{"DataType":"String","StringValue":"nslookup"}}' \
    --message-body "github.com"
{
    "MD5OfMessageBody": "99cd2175108d157588c04758296d1cfc",
    "MD5OfMessageAttributes": "b1badaf4137a97afea1027b635e3ac70",
    "MessageId": "5f24f4a4-ae41-4654-9ab5-acdd2fe5ac28"
}
```

Note that we're using different values for the `Topic` message attribute⸺`figlet` and `nslookup`⸺, and that these correspond to the value of the `topic` annotation we've used above.

Finally, check the logs for `openfaas-sqs-connector` in order to make sure the corresponding functions were invoked:

```shell
$ kubectl -n openfaas logs -l app=openfaas-sqs-connector
(...)
time="2019-09-06T10:36:53Z" level=trace msg="Processing message with id \"daad6027-9503-4952-b1de-4b29314f2048\""
2019/09/06 10:36:53 Invoke function: figlet
2019/09/06 10:36:53 connector-sdk got result: [200] figlet => figlet (270) bytes
[200] figlet => figlet
                         __
  ___  _ __   ___ _ __  / _| __ _  __ _ ___
 / _ \| '_ \ / _ \ '_ \| |_ / _` |/ _` / __|
| (_) | |_) |  __/ | | |  _| (_| | (_| \__ \
 \___/| .__/ \___|_| |_|_|  \__,_|\__,_|___/
      |_|
[openfaas-sqs-connector-5f564d8-mk9h6 openfaas-sqs-connector]
time="2019-09-06T10:36:53Z" level=trace msg="Message successfully processed" message_id=daad6027-9503-4952-b1de-4b29314f2048
time="2019-09-06T10:36:53Z" level=trace msg="Message successfully deleted from the queue" message_id=daad6027-9503-4952-b1de-4b29314f2048
(...)
time="2019-09-06T10:37:11Z" level=trace msg="Processing message with id \"5f24f4a4-ae41-4654-9ab5-acdd2fe5ac28\""
2019/09/06 10:37:11 Invoke function: nslookup
2019/09/06 10:37:11 connector-sdk got result: [200] nslookup => nslookup (78) bytes
[200] nslookup => nslookup
[openfaas-sqs-connector-5f564d8-mk9h6 openfaas-sqs-connector]
Name:      github.com
Address 1: 140.82.118.4 lb-140-82-118-4-ams.github.com
[openfaas-sqs-connector-5f564d8-mk9h6 openfaas-sqs-connector]
time="2019-09-06T10:37:11Z" level=trace msg="Message successfully processed" message_id=5f24f4a4-ae41-4654-9ab5-acdd2fe5ac28
time="2019-09-06T10:37:11Z" level=trace msg="Message successfully deleted from the queue" message_id=5f24f4a4-ae41-4654-9ab5-acdd2fe5ac28
```
