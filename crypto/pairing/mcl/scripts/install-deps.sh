tmpdir=`mktemp -d`

scriptdir=$(cd $(dirname $0); pwd -P)
sourcedir=$(cd $scriptdir/..; pwd -P)
. $scriptdir/shlibs/os.sh

# TODO: This directory should be OS specific
if [ ! -d /usr/local/include/mcl ]; then
(
    cd $tmpdir
    git clone https://github.com/herumi/mcl
    cd mcl/
    git checkout 3130df5 #herumi/mcl v1.52
    cmake -S . -B build -DCMAKE_BUILD_TYPE=Release
    cmake --build build
    sudo cmake --build build --target install
    if [ "$OS" == "Linux" ]; then
        sudo ldconfig
    fi
)
fi

#(
#    cd $sourcedir/
#    GOPATH=$sourcedir go get github.com/stretchr/testify/assert
#    GOPATH=$sourcedir go get github.com/Sirupsen/logrus
#    GOPATH=$sourcedir go get github.com/dustin/go-humanize
#)
