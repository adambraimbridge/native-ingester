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
export|set SRC_UUID_FIELD=uuid,post.uuid,data.uuidv3
export|set DEST_ADDRESS=http://nativerw:8080
export|set DEST_COLLECTIONS_BY_ORIGINS=[{"originId": "http://cmdb.ft.com/systems/methode-web-pub", "collection": "methode"}]
export|set DEST_HEADER=native
./native-ingester[.exe]
```
