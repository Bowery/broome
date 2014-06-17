#!/bin/bash
#
# This script builds the application from source.

# Get the parent directory of where this script is.
set -e

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that directory
cd $DIR
echo $DIR

# Get the git commit
GIT_COMMIT=$(git rev-parse HEAD)
GIT_DIRTY=$(test -n "`git status --porcelain`" && echo "+CHANGES" || true)

# If we're building on Windows, specify an extension
EXTENSION=""
if [ "$(go env GOOS)" = "windows" ]; then
    EXTENSION=".exe"
fi

GOPATHSINGLE=${GOPATH%%:*}
if [ "$(go env GOOS)" = "windows" ]; then
    GOPATHSINGLE=${GOPATH%%;*}
fi

if [ "$(go env GOOS)" = "freebsd" ]; then
  export CC="clang"
  export CGO_LDFLAGS="$CGO_LDFLAGS -extld clang" # Workaround for https://code.google.com/p/go/issues/detail?id=6845
fi

# Install dependencies
echo "--> Installing dependencies to speed up builds..."
go get \
  -ldflags "${CGO_LDFLAGS}" \
  ./...

# Build Broome!
echo "--> Building Broome..."
cd "${DIR}"
go build \
    -ldflags "${CGO_LDFLAGS} -X main.GitCommit ${GIT_COMMIT}${GIT_DIRTY}" \
    -v \
    -o ../bin/broome${EXTENSION}
cp ../bin/broome${EXTENSION} ${GOPATHSINGLE}/bin

/bin/bash ${DIR}/scripts/check-mongo.sh
/bin/bash ${DIR}/scripts/check-myth.sh

echo "--> Compiling Assets with Myth"
myth static/style.css static/out.css


echo "--> Running on Port 4000"
ENV=development ../bin/broome
