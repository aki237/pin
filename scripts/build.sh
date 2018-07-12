DATE=$(echo \"main.buildDate=$(date +"%H:%M %d-%b-%Y %Z")\")
VERSION=$(echo \"main.version=$(cat VERSION).$(git rev-parse --short HEAD)\")
build () {
    go build -ldflags "-w -s -X $DATE -X $VERSION" -o $1
}

export CGO_ENABLED=0
BIN_FILE_NAME_PREFIX=$1
PROJECT_DIR=$2
PLATFORMS="linux/amd64 linux/arm \
	   darwin/amd64 \
	   freebsd/amd64 freebsd/arm \
	   netbsd/amd64 netbsd/arm \
	   openbsd/amd64 openbsd/arm \
	   dragonfly/amd64"

for PLATFORM in $PLATFORMS; do
    export GOOS=${PLATFORM%/*}
    export GOARCH=${PLATFORM#*/}
    echo "Building for $GOOS $GOARCH"
    FILEPATH="$PROJECT_DIR/artifacts/${GOOS}-${GOARCH}"
    mkdir -p $FILEPATH
    BIN_FILE_NAME="$FILEPATH/${BIN_FILE_NAME_PREFIX}"
    build $BIN_FILE_NAME || FAILURES="${FAILURES} ${PLATFORM}"
    cp Readme.md LICENSE $FILEPATH
    tar czf $FILEPATH/pin-${GOOS}-${GOARCH}.tar.gz -C $FILEPATH pin Readme.md LICENSE
    rm $FILEPATH/pin $FILEPATH/Readme.md $FILEPATH/LICENSE
done
