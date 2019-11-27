#!/bin/bash

set -eux -o pipeline

conffiles=${@}
if [ "$conffiles" = "" ] ;then echo "invalid arguments"; exit 1; fi

for conffile in $conffiles; do
  if [ ! -e $conffile ]; then echo "no such files"; exit 1; fi

  nsupdate <(echo "server 127.0.0.1")
  while read line; do
    nsupdate $line
  done < $conffile
  nsupdate <(echo "send")

done
