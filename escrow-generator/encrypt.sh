#!/bin/bash

#set -x

OUTPUT_DIR=$1
PRIVATE_KEY_PATH=$2
TARGET_KEYS_RAW="${@:3}"

PUB_KEYS=$(echo $TARGET_KEYS_RAW | tr " " "\n")

GPG_PATH=`which gpg`
GPG_HOMEDIR="./gpg"
GPG_CMD="$GPG_PATH --homedir $GPG_HOMEDIR --trust-model always  -v"

CHMOD_PATH=`which chmod`
MKDIR_PATH=`which mkdir`

$MKDIR_PATH $GPG_HOMEDIR
$CHMOD_PATH 0700 $GPG_HOMEDIR


echo "use-agent" > $GPG_HOMEDIR/gpg.conf
echo "pinentry-mode loopback" > $GPG_HOMEDIR/gpg.conf
echo "allow-loopback-pinentry" > $GPG_HOMEDIR/gpg-agent.conf

echo $OUTPUT_DIR
ls -alh $OUTPUT_DIR

$GPG_CMD --batch --yes --passphrase-fd 0 --import $PRIVATE_KEY_PATH < .gpgpass
if [ $? -ne 0 ]; then
  echo "ERROR: Could not load private key"
  exit 1
fi

$GPG_CMD --list-keys
if [ $? -ne 0 ]; then
  echo "ERROR: Could not list gpg keys"
  exit 1
fi

RECV=""
for PUB_KEY in $PUB_KEYS
do
  $GPG_CMD --import ./$PUB_KEY.asc
  if [ $? -ne 0 ]; then
    echo "ERROR: Could not import public key"
    exit 1
  fi
  RECV="$RECV -r $PUB_KEY "
done

echo $RECV

$GPG_CMD --list-keys

FILES=$(ls $OUTPUT_DIR | grep csv.gz)

for FILE in $FILES
do
  $GPG_CMD --batch --yes --passphrase-fd 0 --encrypt --sign --armor $RECV $OUTPUT_DIR/$FILE < .gpgpass
  if [ $? -ne 0 ]; then
    echo "ERROR: encrypting $FILE"
    exit 1
  fi
  rm $OUTPUT_DIR/$FILE
done

rm -rf $GPG_HOMEDIR
