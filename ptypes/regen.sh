#!/bin/bash
#
# This script fetches and rebuilds the "well-known types" protocol buffers.
# To run this you will need protoc and goprotobuf installed;
# see https://github.com/golang/protobuf for instructions.
# You also need Go and Git installed.

set -Ee

PKG=github.com/golang/protobuf/ptypes
UPSTREAM=https://github.com/google/protobuf
UPSTREAM_SUBDIR=src/google/protobuf
PROTO_FILES='
  any.proto
  duration.proto
  empty.proto
  struct.proto
  timestamp.proto
  wrappers.proto
'

function die() {
  echo 1>&2 $*
  exit 1
}

function which_sed() {
  local uname_s="$(uname -s)"
  case "${uname_s}" in
    Darwin)
      if ! which gsed > /dev/null; then
        die "must install gsed on a mac, try brew install gnu-sed"
      fi
      echo "gsed"
      ;;
    Linux)
      if ! which sed > /dev/null; then
        die "cannot find sed"
      fi
      echo "sed"
      ;;
    *)
      die "unknown result from uname -s: ${uname_s}"
  esac
}

# Sanity check that the right tools are accessible.
for tool in go git protoc protoc-gen-go; do
  q=$(which $tool) || die "didn't find $tool"
  echo 1>&2 "$tool: $q"
done

tmpdir=$(mktemp -d -t regen-wkt.XXXXXX)
trap 'rm -rf $tmpdir' EXIT

echo -n 1>&2 "finding package dir... "
pkgdir=$(go list -f '{{.Dir}}' $PKG)
echo 1>&2 $pkgdir
base=$(echo $pkgdir | $(which_sed) "s,/$PKG\$,,")
echo 1>&2 "base: $base"
cd $base

echo 1>&2 "fetching latest protos... "
git clone -q $UPSTREAM $tmpdir
# Pass 1: build mapping from upstream filename to our filename.
declare -A filename_map
for f in $(cd $PKG && find * -name '*.proto'); do
  echo -n 1>&2 "looking for latest version of $f... "
  up=$(cd $tmpdir/$UPSTREAM_SUBDIR && find * -name $(basename $f) | grep -v /testdata/)
  echo 1>&2 $up
  if [ $(echo $up | wc -w) != "1" ]; then
    die "not exactly one match"
  fi
  filename_map[$up]=$f
done
# Pass 2: copy files, making necessary adjustments.
for up in "${!filename_map[@]}"; do
  f=${filename_map[$up]}
  shortname=$(basename $f | $(which_sed) 's,\.proto$,,')
  cat $tmpdir/$UPSTREAM_SUBDIR/$up |
    # Adjust proto package.
    # TODO(dsymonds): Upstream the go_package option instead.
    $(which_sed) '/^package /a option go_package = "'github.com\/golang\/protobuf\/ptypes\/${shortname}'";' |
    # Unfortunately "package struct" doesn't work.
    $(which_sed) '/option go_package/s,struct",struct;structpb",' |
    cat > $PKG/$f
done

# Run protoc once per package.
#
# dirname does not work with multiple arguments on Darwin, so we run the risk
# of protoc being invoked multiple times per directory, which isn't an issue here
# and also does not happen
for file in $(find $PKG -name '*.proto'); do
  dir=$(dirname $file)
  echo 1>&2 "* $dir"
  protoc --go_out=. $dir/*.proto
done
echo 1>&2 "All OK"
