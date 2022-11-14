#!/bin/bash

# Apache License Version 2.0, January 2004
# https://github.com/debeando/mysql2parquet/blob/master/LICENSE

set -e

if [[ "${OSTYPE}" == "linux"* ]]; then
  FILE="mysql2parquet-linux_amd64.tar.gz"
elif [[ "$OSTYPE" == "darwin"* ]]; then
  FILE="mysql2parquet-darwin_amd64.tar.gz"
else
  echo "Only works on Linux or Darwin amd64."
  exit
fi

if [[ $EUID -ne 0 ]]; then
    echo "$0 is not running as root. Try using sudo."
    exit 2
fi

if ! type "wget" &> /dev/null; then
  echo "The program 'wget' is currently not installed, please install it to continue."
  exit
fi

TAG=$(wget -qO- "https://api.github.com/repos/debeando/mysql2parquet/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -f /usr/local/bin/mysql2parquet ]; then
  rm -f /usr/local/bin/mysql2parquet
fi

if [ -L /usr/bin/mysql2parquet ]; then
  rm -f /usr/bin/mysql2parquet
fi

if [ -n "${FILE}" ]; then
  wget -qO- "https://github.com/debeando/mysql2parquet/releases/download/${TAG}/${FILE}" | tar xz -C /usr/local/bin/
fi

if [[ "${OSTYPE}" == "linux"* ]]; then
  if [ -f /usr/local/bin/mysql2parquet ]; then
    ln -s /usr/local/bin/mysql2parquet /usr/bin/mysql2parquet
  fi
fi
