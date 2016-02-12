FROM alpine:3.3

ADD . /native-ingester/

RUN apk add --update bash \
  && apk --update add git bzr \
  && apk --update add go \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/native-ingester" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && cp -r native-ingester/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && go get -t ./... \
  && go build \
  && mv native-ingester /app \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/*

CMD [ "/app" ] 