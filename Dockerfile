FROM alpine:3.5

COPY . .git /native-ingester/

RUN apk --update add git go libc-dev ca-certificates \
  && export GOPATH=/gopath \
  && REPO_PATH="github.com/Financial-Times/native-ingester" \
  && mkdir -p $GOPATH/src/${REPO_PATH} \
  && cp -r native-ingester/* $GOPATH/src/${REPO_PATH} \
  && cd $GOPATH/src/${REPO_PATH} \
  && BUILDINFO_PACKAGE="github.com/Financial-Times/service-status-go/buildinfo." \
  && VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && LDFLAGS="-X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && echo $LDFLAGS \
  && go get -u github.com/kardianos/govendor \
  && $GOPATH/bin/govendor sync \
  && go get -t -v ./... \
  && go build -ldflags="${LDFLAGS}" \
  && mv native-ingester /native-ingester-app \
  && apk del go git bzr \
  && rm -rf $GOPATH /var/cache/apk/* /native-ingester

CMD [ "/native-ingester-app" ]
