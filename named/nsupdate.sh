#!/bin/bash

# TODO dockerize
sleep 30

conffiles=${@}
if [ "$conffiles" = "" ] ;then echo "invalid arguments"; exit 1; fi

for conffile in $conffiles; do
  if [ ! -e $conffile ]; then echo "no such files"; exit 1; fi

  nsupdate <(cat << _EOF_
server 127.0.0.1
$(while read line; do
echo "update add $line"
done < $conffile)
send
_EOF_
)

done
