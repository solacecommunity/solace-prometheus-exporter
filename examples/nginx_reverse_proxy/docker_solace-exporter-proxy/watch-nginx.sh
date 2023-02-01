#!/bin/bash
#

config_file="/etc/nginx/nginx.conf"

wait_file() {
    local file="$1"; shift
    local wait_seconds="${1:-10}"; shift # 10 seconds as default timeout

    until test $((wait_seconds--)) -eq 0 -o -f "$file" ; do sleep 1; done

    ((++wait_seconds))
}

wait_file "$config_file" 90 || {
    echo "Missing $config_file. Giving up after waiting 90sec."
    exit 1
}

{
  echo "Starting nginx..."
  nginx "$@" && exit 1
} &

md5=`md5sum $config_file | awk '{ print $1 }'`


while true; do
  nextMd5=`md5sum $config_file | awk '{ print $1 }'`
  if [[ "$nextMd5" != "$md5" ]]; then
    echo "Try to verify updated nginx config..."
    nginx -t
    if [ $? -ne 0 ]; then
      echo "ERROR: New configuration is invalid!!"
    else
      echo "Reloading nginx with new config..."
      nginx -s reload
    fi

     md5=$nextMd5
  fi;
  sleep 3   
done