#!/usr/bin/env bash

# error codes
# 0 - exited without problems
# 1 - parameters not supported were used or some unexpected error occurred
# 2 - OS not supported by this script
# 3 - installed version of oneshot is up to date

set -e

usage() { echo "Usage: curl https://github.com/raphaelreyna/oneshot/raw/master/install.sh | sudo bash " 1>&2; exit 1; }

#create tmp directory and move to it with macOS compatibility fallback
tmp_dir=`mktemp -d 2>/dev/null || mktemp -d -t 'oneshot-install.XXXXXXXXXX'`; cd $tmp_dir

#check installed version of oneshot to determine if update is necessary
version=`oneshot -v 2>>errors | head -n 1 | awk '{print $4}'`
current_version=`curl -s -L https://github.com/raphaelreyna/oneshot/raw/master/version.txt`
if [ "$version" = "$current_version" ]; then
    printf "\nThe latest version of oneshot ${version} is already installed.\n\n"
    exit 3
fi

#detect the platform
OS="`uname`"
case $OS in
  Linux)
    OS='linux'
    ;;
  Darwin)
    OS='osx'
    ;;
  *)
    echo 'OS not supported'
    exit 2
    ;;
esac

ARCH_TYPE="`uname -m`"
case $ARCH_TYPE in
  x86_64|amd64)
    ARCH_TYPE='x86_64'
    ;;
  i?86|x86)
    ARCH_TYPE='386'
    ;;
  arm*)
    ARCH_TYPE='arm'
    ;;
  aarch64)
    ARCH_TYPE='arm64'
    ;;
  *)
    echo 'OS type not supported'
    exit 2
    ;;
esac

#download and untar
download_link="https://github.com/raphaelreyna/oneshot/releases/download/${current_version}/oneshot_${current_version}.${OS}-${ARCH_TYPE}.tar.gz"
oneshot_tarball="oneshot_${current_version}.${OS}-${ARCH_TYPE}.tar.gz"

curl -s -O -L $download_link
untar_dir="oneshot_untar"
mkdir $untar_dir
tar -xzf $oneshot_tarball -C $untar_dir
cd $untar_dir

#install oneshot
case $OS in
  'linux')
    cp oneshot /usr/bin/oneshot.new
    chmod 755 /usr/bin/oneshot.new
    chown root:root /usr/bin/oneshot.new
    mv /usr/bin/oneshot.new /usr/bin/oneshot
    ;;
  'osx')
    mkdir -p /usr/local/bin
    cp oneshot /usr/local/bin/oneshot.new
    mv /usr/local/bin/oneshot.new /usr/local/bin/oneshot
    ;;
  *)
    echo 'OS not supported'
    exit 2
esac


# Let user know oneshot was installed
version=`oneshot --version 2>>errors | head -n 1 | awk '{print $4}'`

printf "\n${version} has successfully installed.\n"
printf 'You may now run "oneshot -h" for help with using oneshot.\n'
printf 'Visit https://github.com/raphaelreyna/oneshot for more information.\n\n'
exit 0
