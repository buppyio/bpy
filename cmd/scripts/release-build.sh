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

if test -z $CURTAG
then
	echo "expected a tag"
	exit 1
fi

if !test $CURTAG != $BPYVERSION
then
	echo "the current version must match the current tag"
	exit 1
fi

releasebuild () {
	export GOOS=$1
	export GOARCH=$2
	export RELEASEDIR=bpy_release/bpy-$BPYVERSION-$GOOS-$GOARCH
	export ARCHIVEDIR=bpy_release/archives
	export TARPATH=$ARCHIVEDIR/bpy-$BPYVERSION-$GOOS-$GOARCH.tar.gz
	export ZIPPATH=$ARCHIVEDIR/bpy-$BPYVERSION-$GOOS-$GOARCH.zip
	
	mkdir -p $RELEASEDIR
	go build -o $RELEASEDIR/bpy github.com/buppyio/bpy/cmd/bpy
	
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

