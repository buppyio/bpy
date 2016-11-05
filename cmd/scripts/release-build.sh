set -e

export BPYVERSION=$1

if test -z $BPYVERSION
then
	echo "usage: release-build.sh VERSION"
	exit 1
fi

WORKINGDIR=`pwd`
cd $GOPATH/src/github.com/buppyio/bpy/
CURTAG=`git tag`
cd $WORKINGDIR

releasebuild () {
	export GOOS=$1
	export GOARCH=$2
	RELEASEDIR=bpy_release/bpy-$BPYVERSION-$GOOS-$GOARCH
	ARCHIVEDIR=bpy_release/archives
	TARPATH=$ARCHIVEDIR/bpy-$BPYVERSION-$GOOS-$GOARCH.tar.gz
	ZIPPATH=$ARCHIVEDIR/bpy-$BPYVERSION-$GOOS-$GOARCH.zip
	BPYCOMMIT=`git rev-parse HEAD`
	
	mkdir -p $RELEASEDIR
	go build -ldflags "-X github.com/buppyio/bpy/cmd/bpy/version.BpyVersion=$BPYVERSION -X github.com/buppyio/bpy/cmd/bpy/version.BpyCommit=$BPYCOMMIT" -o $RELEASEDIR/bpy github.com/buppyio/bpy/cmd/bpy
	
	if test $GOOS = windows
	then
		mv $RELEASEDIR/bpy $RELEASEDIR/bpy.exe
	fi
	
	mkdir -p $ARCHIVEDIR
	tar cpzf $TARPATH $RELEASEDIR
	zip $ZIPPATH $RELEASEDIR/*
}

releasebuild linux amd64
releasebuild openbsd amd64
releasebuild freebsd amd64
releasebuild windows amd64

