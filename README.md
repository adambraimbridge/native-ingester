Native Ingester
===============
[![CircleCI](https://circleci.com/gh/Financial-Times/native-ingester.svg?style=svg)](https://circleci.com/gh/Financial-Times/native-ingester) [![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/native-ingester)](https://goreportcard.com/report/github.com/Financial-Times/native-ingester) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/native-ingester/badge.svg)](https://coveralls.io/github/Financial-Times/native-ingester)

Native ingester implements the following functionality:
1. It consumes messages containing native CMS content or native CMS metadata from ONE queue topic.
1. According to the data source, native ingester writes the content or metadata to a specific db collection.
1. Optionally, it forwards consumed messages to a different queue.

## Installation & running locally
[dep](https://github.com/golang/dep/) is a pre-requisite.
Installation:
```
go get github.com/Financial-Times/native-ingester
cd $GOPATH/src/github.com/Financial-Times/native-ingester
dep ensure -vendor-only
go test ./...
go install

```
Run it locally:
```
$GOPATH/bin/native-ingester
    --read-queue-addresses={source-zookeeper-address}
    --read-queue-group={topic-group}
    --read-queue-topic={source-topic}
    --source-uuid-field='["uuid", "post.uuid", "data.uuidv3"]'
    --native-writer-address={native-writer-address}
```
List all the possible options:
```
$  native-ingester -h

Usage: native-ingester [OPTIONS]

A service to ingest native content of any type and persist it in the native store, then, if required forwards the message to a new message queue

Options:
  --port="8080"                                 Port to listen on ($PORT)
  --read-queue-addresses=[]                     Zookeeper addresses (host:port) to connect to the consumer queue. ($Q_READ_ADDR)
  --read-queue-group=""                         Group used to read the messages from the queue. ($Q_READ_GROUP)
  --read-queue-topic=""                         The topic to read the messages from. ($Q_READ_TOPIC)
  --native-writer-address=""                    Address (URL) of service that writes persistently the native content ($NATIVE_RW_ADDRESS)
  --config="config.json"                        Configuration file - Mapping from (originId (URI), Content Type) to native collection name, in JSON format, for content_type attribute specify a RegExp Literal expression.
  --content-uuid-fields=[]                      List of JSONPaths that point to UUIDs in native content bodies. e.g. uuid,post.uuid,data.uuidv3 ($NATIVE_CONTENT_UUID_FIELDS)
  --write-queue-address=""                      Kafka address (host:port) to connect to the producer queue. ($Q_WRITE_ADDR)
  --write-topic=""                              The topic to write the messages to. ($Q_WRITE_TOPIC)
  --content-type="Content"                      The type of the content (for logging purposes, e.g. "Content" or "Annotations") the application is able to handle. ($CONTENT_TYPE)
  --appName="native-ingester"                   The name of the application ($APP_NAME)
```

## Admin endpoints

  - `https://{host}/__native-store-{type}/__health`
  - `https://{host}/__native-store-{type}/__gtg`

Note: All API endpoints in CoCo require Authentication.
See [service run book](https://dewey.ft.com/native-ingester.html) on how to access cluster credentials.  
