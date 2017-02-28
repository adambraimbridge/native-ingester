Native Ingester
===============
[![CircleCI](https://circleci.com/gh/Financial-Times/native-ingester.svg?style=svg)](https://circleci.com/gh/Financial-Times/native-ingester) [![Go Report Card](https://goreportcard.com/badge/github.com/Financial-Times/native-ingester)](https://goreportcard.com/report/github.com/Financial-Times/native-ingester) [![Coverage Status](https://coveralls.io/repos/github/Financial-Times/native-ingester/badge.svg?branch=master)](https://coveralls.io/github/Financial-Times/native-ingester?branch=master) [![codecov](https://codecov.io/gh/Financial-Times/native-ingester/branch/master/graph/badge.svg)](https://codecov.io/gh/Financial-Times/native-ingester)

Native ingester implements the following functionality:
1. It consumes messages containing native CMS content or native CMS metadata from ONE queue topic.
1. According to the data source, native ingester writes the content or metadata to a specific db collection.
1. Optionally, it forwards consumed messages to a different queue.

## Installation & running locally
Installation:
```
go get -u github.com/kardianos/govendor
go get github.com/Financial-Times/native-ingester
cd $GOPATH/src/github.com/Financial-Times/native-ingester
govendor sync 
go test ./...
go install

```
Run it locally:
```
$GOPATH/bin/native-ingester
    --read-queue-addresses={source-kafka-proxy-address}
    --read-queue-group={topic-group}
    --read-queue-topic={source-topic}
    --read-queue-host-heade={source-kafka-proxy-host-header}
    --source-concurrent-processing='true'
    --source-uuid-field='["uuid", "post.uuid", "data.uuidv3"]'
    --native-writer-address={native-writer-address}
    --native-writer-host-header={native-writer-host-header}
```
List all the possible options:
```
$ native-ingester -v

Usage: native-ingester [OPTIONS]

A service to ingest native content of any type and persist it in the native store, then, if required forwards the message to a new message queue

Options:
  --read-queue-addresses=[]                     Addresses to connect to the consumer queue (URLs). ($Q_READ_ADDR)
  --read-queue-group=""                         Group used to read the messages from the queue. ($Q_READ_GROUP)
  --read-queue-topic=""                         The topic to read the meassages from. ($Q_READ_TOPIC)
  --read-queue-host-header="kafka"              The host header for the queue to read the meassages from. ($Q_READ_QUEUE_HOST_HEADER)
  --read-queue-concurrent-processing=false      Whether the consumer uses concurrent processing for the messages ($Q_READ_CONCURRENT_PROCESSING)
  --native-writer-address=""                    Address of service that writes persistently the native content ($NATIVE_RW_ADDRESS)
  --native-writer-collections-by-origins="[]"   Map in a JSON-like format. originId referring the collection that the content has to be persisted in. e.g. [{"http://cmdb.ft.com/systems/methode-web-pub":"methode"}] ($NATIVE_RW_COLLECTIONS_BY_ORIGINS)
  --native-writer-host-header="nativerw"        coco-specific header needed to reach the destination address ($NATIVE_RW_HOST_HEADER)
  --content-uuid-fields=[]                      List of JSONPaths that point to UUIDs in native content bodies. e.g. uuid,post.uuid,data.uuidv3 ($NATIVE_CONTENT_UUID_FIELDS)
  --write-queue-address=""                      Address to connect to the producer queue (URL). ($Q_WRITE_ADDR)
  --write-topic=""                              The topic to write the meassages to. ($Q_WRITE_TOPIC)
  --write-queue-host-header="kafka"             The host header for the queue to write the meassages to. ($Q_WRITE_QUEUE_HOST_HEADER)

```

## Deployment to CoCo
- This service should be deployed in the publishing clusters.
- Read detailed explanation of the [CoCo Environments] (https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/coco/environments)
- Follow instructions on how to deploy outlined in the [Coco Deployment Process document] (https://sites.google.com/a/ft.com/technology/systems/dynamic-semantic-publishing/coco/deploy-process)

#### Steps:
- update version in [github/FinancialTimes/pub-service-files/services.yaml] (https://github.com/Financial-Times/pub-service-files/blob/master/services.yaml)  

- if necessary update:
     - [native-ingester-cms@.service] (https://github.com/Financial-Times/up-service-files/blob/master/native-ingester-cms%40.service)
     - [native-ingester-cms-sidekick@.service] (https://github.com/Financial-Times/up-service-files/blob/master/native-ingester-cms-sidekick%40.service)
     - [native-ingester-metadata@.service] (https://github.com/Financial-Times/up-service-files/blob/master/native-ingester-metadata%40.service)
     - [native-ingester-metadata-sidekick@.service] (https://github.com/Financial-Times/up-service-files/blob/master/native-ingester-metadata-sidekick%40.service)

## Admin endpoints (CoCo)

  - `https://{host}/__native-store-{type}/__health`
  - `https://{host}/__native-store-{type}/__gtg`

Note: All API endpoints in CoCo require Authentication.
See [service run book](https://dewey.ft.com/native-ingester.html) on how to access cluster credentials.  
