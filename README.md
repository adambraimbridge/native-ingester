# Native Ingester
* Ingests native content (articles, metadata, etc.) and persists them in the native-store.
* Reads from ONE queue topic, writes to ONE db collection.
# How to run

```
export|set SRC_ADDR=http://kafka:8080
export|set SRC_GROUP=FooGroup
export|set SRC_TOPIC=FooBarEvents
export|set SRC_QUEUE=kafka
export|set SRC_CONCURRENT_PROCESSING=true
export|set DEST_ADDRESS=http://nativerw:8080
export|set DEST_COLLECTION=DestCollection
./native-ingester[.exe]
```
