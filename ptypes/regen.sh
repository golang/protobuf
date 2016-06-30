#!/bin/bash
# This script fetches and rebuilds the "well-known types" protocol buffers.
# To run this you will need protoc and goprotobuf installed;
# see https://github.com/golang/protobuf for instructions.
# You also need Go and Git installed.

set -Ee

PKG=github.com/gogo/protobuf/ptypes
UPSTREAM=https://github.com/google/protobuf
UPSTREAM_SUBDIR=src/google/protobuf

function die() {
  echo 1>&2 $*
  exit 1
}

# Sanity check that the right tools are accessible.
for tool in go git protoc protoc-gen-go; do
  q=$(which $tool) || die "didn't find $tool"
  echo 1>&2 "$tool: $q"
done

# Can be use for tests of regen2.sh
# tmpdir=/tmp/upstream
tmpdir=$(mktemp -d -t regen-wkt.XXXXXX)
git clone -q $UPSTREAM $tmpdir
trap 'rm -rf $tmpdir' EXIT

# Jump to the working directory
pushd $GOPATH/src/$PKG &>/dev/null

# Pass 1: sanitizing
for F in $(find . -name '*.proto'); do

  inst=$(find $tmpdir/$UPSTREAM_SUBDIR -name $(basename $F) -and -not -path "*/testdata/*"  -print)

  if [ $(echo "$inst" | wc -l) -ne 1 ] ; then
    die "Did not find exactly one instance of '$F' in '$tmpdir/$UPSTREAM_SUBDIR'!"
  fi

done

# Pass 2: copy and modify
# We are sure the upstream is in valid state as per pass 1
for F in $(find . -name '*.proto'); do
  shortname=$(expr $(basename $F) : '\(.*\)\.proto')

  # Unfortunately "package struct" doesn't work.
  # Handle the special case here instead of passing all files through sed
  # a second time
  if [ $shortname == "struct" ] ; then
    shortname="structpb"
  fi

  fn="$tmpdir/$UPSTREAM_SUBDIR/$(basename $F)"
  # Upstream now seems to have to go_package option
  sed -e "s/^\(option go_package\).*=.*/\1 = \"$shortname\";/" "$fn" > $F
done

# Compile
for dir in $(find . -name '*.proto' -exec dirname {} \; | sort -u); do
  echo -en "* $dir... " 1>&2
  protoc --go_out=. $dir/*.proto
  if [ $? -ne 0 ]; then
    die "Error creating output files"
  fi
  echo "...Success!" 1>&2
done

# Jump back to the original directory
popd &>/dev/null
