# UPP - Native Metadata Ingester

The Native Metadata ingester consumes messages from a Kafka topic and propagates them to be written by the read-write service in the Native store.

## Primary URL

<https://upp-prod-publish-glb.upp.ft.com/__native-ingester-metadata/>

## Service Tier

Platinum

## Lifecycle Stage

Production

## Delivered By

content

## Supported By

content

## Known About By

- hristo.georgiev
- robert.marinov
- elina.kaneva
- tsvetan.dimitrov
- elitsa.pavlova
- ivan.nikolov
- miroslav.gatsanoga
- kalin.arsov
- mihail.mihaylov
- dimitar.terziev

## Host Platform

AWS

## Architecture

The Native Metadata Ingester consumes messages containing native CMS metadata from the "PreNativeCmsMetadataPublicationEvents" Kafka topic. According to the data source, it sends the metadata to be written to a specific DB collection. Optionally, it forwards consumed messages to a different queue.

## Contains Personal Data

No

## Contains Sensitive Data

No

## Dependencies

- nativestorereaderwriter
- upp-kafka
- upp-zookeeper

## Failover Architecture Type

ActivePassive

## Failover Process Type

FullyAutomated

## Failback Process Type

FullyAutomated

## Failover Details

The service is deployed in both publishing clusters. The failover guide for the cluster is located here:
<https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/publishing-cluster>

## Data Recovery Process Type

NotApplicable

## Data Recovery Details

The service does not store data, so it does not require any data recovery steps.

## Release Process Type

PartiallyAutomated

## Rollback Process Type

Manual

## Release Details

Manual failover is needed when a new version of the service is deployed to production. Otherwise, an automated failover is going to take place when releasing.
For more details about the failover process please see: <https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/publishing-cluster>

## Key Management Process Type

Manual

## Key Management Details

To access the service clients need to provide basic auth credentials.
To rotate credentials you need to login to a particular cluster and update varnish-auth secrets.

## Monitoring

Publishing EU health:
- <https://upp-prod-publish-eu.upp.ft.com/__health/__pods-health?service-name=native-ingester-metadata>

Publishing US health:
- <https://upp-prod-publish-us.upp.ft.com/__health/__pods-health?service-name=native-ingester-metadata>

## First Line Troubleshooting

<https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting>

## Second Line Troubleshooting

Please refer to the GitHub repository README for troubleshooting information.
