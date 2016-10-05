Native Ingester
===============

* Ingests native content (articles, metadata, etc.) and persists them in the native-store.
* Reads from ONE queue topic, adds a timestamp to the message and writes to ONE db collection.

## Installation & running locally
* `go get -u github.com/Financial-Times/native-ingester`
* `cd $GOPATH/src/github.com/Financial-Times/native-ingester`
* `go test ./...`
* `go install`
* `$GOPATH/bin/native-ingester --neo-url={neo4jUrl} --port={port}`
* `$GOPATH/bin/native-ingester --source-addresses={source-addresses} --source-group={source-group} 
    --source-topic={source-topic} --source-queue={source-queue} --source-concurrent-processing='true' 
    --source-uuid-field='uuid' --destination-address={destination-address}`

--source-addresses is the url of the queue/proxy to listen to
--source-group is the groupname to use when listen to the queue
--source-topic is the topic to listen to on the queue
--source-queue is the name of the queue
--source-concurrent-processing allows processing multiple messages at one time
--source-uuid-field is the field which represents uuid in the message
--destination-address is the url of native-rw for writing to native store
--destination-collections-by-origins is for whitelisting messages by origin-id
--destination-header is a coco header for routing
 

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
See [service run book] (https://sites.google.com/a/ft.com/ft-technology-service-transition/home/run-book-library/native-ingester) on how to access cluster credentials.  

