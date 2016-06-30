#!/bin/bash
# This script fetches and rebuilds the "well-known types" protocol buffers.
# To run this you will need protoc and goprotobuf installed;
# see https://github.com/golang/protobuf for instructions.
# You also need Go and Git installed.
#
# This is a complete rewrite of the original regen.sh in order to address #196
# and reduce the reliance on external tools (which might behave differently
# from system to system) to the bare minimum.

# Set here to make sure it gets acknowledged even
# in the weirdest of circumstances
set -Ee

PKG=github.com/golang/protobuf/ptypes
UPSTREAM=https://github.com/google/protobuf
UPSTREAM_SUBDIR=src/google/protobuf


function die() {
  echo 1>&2 $*
  exit 1
}

# Sanity check that the right tools are accessible.
for tool in go git protoc protoc-gen-go find; do
  q=$(which $tool) || die "didn't find $tool"
  echo 1>&2 "$tool: $q"
done

# Can be use for tests of regen2.sh
#tmpdir=/tmp/upstream
tmpdir=$(mktemp -d -t regen-wkt.XXXXXX)
git clone $UPSTREAM $tmpdir
trap 'rm -rf $tmpdir' EXIT

# Jump to the working directory
# It has to be $GOPATH/src/$PKG, because
# * that is where the sources are supposed to be
# * protoc will not find the data necessary to import
# TODO: this is arguable. Maybe we should just have a basepath.
pushd $GOPATH/src/$PKG &>/dev/null
# Jump back to the original directory
trap 'popd &>/dev/null' EXIT

# Pass 1: sanitizing
for F in $(find . -type f -name '*.proto'); do

  count=$(find $tmpdir/$UPSTREAM_SUBDIR \
    -type f -name $(expr "$F" : '.*/\(.*\.proto\)') \
    -and -not -path "*/testdata/*" | wc -l)

  if [ $count -ne 1 ] ; then
    die "Did not find exactly one instance of '$F' in '$tmpdir/$UPSTREAM_SUBDIR', found $count!"
  fi

done

# Pass 2: copy the protofiles to their according location
# We are sure the upstream is in valid state as per pass 1
# Upstream now has the go_package option set, so no need for
# mangling with names any more
for F in $(find . -name '*.proto'); do
  cp  "$tmpdir/$UPSTREAM_SUBDIR/$(expr "$F" : '.*/\(.*.proto\)')" $F
  if [ $? -ne 0 ];then
    die "Error while copying $F"
  fi
done

# Compile
for dir in $(find -type f -name *.proto -exec dirname {} \; | sort -u); do
  echo -en "* $dir... " 1>&2
  protoc --go_out=$GOPATH/src $dir/*.proto
  if [ $? -ne 0 ]; then
    die "Error creating output files"
  fi
  echo "...Success!" 1>&2
done

