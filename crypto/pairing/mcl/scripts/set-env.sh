#!/bin/bash
    
scriptdir=$(cd $(dirname ${BASH_SOURCE[@]}); pwd -P)
sourcedir=$(cd $scriptdir/..; pwd -P)

#export GOPATH=$sourcedir
#echo "Set \$GOPATH = $GOPATH"

# Unless we set this, go caches test results, which can lead to missed bugs in randomized tests.
export GOFLAGS="-count=1"
echo "Set \$GOFLAGS = $GOFLAGS"

#export PATH="$PATH:$scriptdir"
#echo "Set \$PATH = $PATH"
