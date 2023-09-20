#!/bin/bash

#set -x

ESCROWUSERNAME=rde-user
ESCROWHOST=rde.example.com
ESCROWPUBKEYS="5063CD06 E39C7BDB"
ESCROWPRIVKEYPATH="./secret/rde.priv.asc"


rm output/*
rmdir output
$HOME/bin/escrow-generator/escrow-generator -conf prod.conf

./encrypt.sh ./output/ $ESCROWPRIVKEYPATH $ESCROWPUBKEYS
if [ $? -ne 0 ]; then
    echo "ERROR: encrypt.sh did not exit cleanly"
    exit 1
fi

rm ./output/batch

FILES=`ls ./output/`

touch ./output/batch

for FILE in $FILES; do
    echo "put $FILE" >> ./output/batch
done 

echo "bye" >> ./output/batch

cd output
cat ../.sshpass 
/usr/bin/sftp -oBatchMode=no -b ./batch $ESCROWUSERNAME@$ESCROWHOST:
