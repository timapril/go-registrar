#!/bin/bash

usage(){
	echo "Usage: $0 certificate-path plaintext-passphrase"
	exit 1
}

if [ "$#" -ne 2 ];
  then echo "Not enough args"
fi

CERT_PATH=$1
PASSPHRASE_PATH=$2

OPENSSL_PATH=`which openssl`
CAT_PATH=`which cat`
BASE64_PATH=`which base64`

DATE=`date +%s`

KEY_PATH="./tmp.$DATE.key"
ENC_PASSPHRASE_PATH="./enc.pass.b64"

# Phase 1: Generate the encryption key
$OPENSSL_PATH x509 -pubkey -noout -in $CERT_PATH > $KEY_PATH

# Phase 2: Encrypt the passphrase
$CAT_PATH $PASSPHRASE_PATH | $OPENSSL_PATH rsautl -encrypt -pubin -inkey $KEY_PATH | $BASE64_PATH | tr -d '\n' > $ENC_PASSPHRASE_PATH
# Cleanup
rm $KEY_PATH
