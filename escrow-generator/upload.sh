#!/usr/bin/expect -f

ESCROWUSERNAME=rde-user
ESCROWHOST=rde.example.com

sshPass=`cat ./.sshpass`

spawn scp -r ./output/ $ESCROWUSERNAME@$ESCROWHOST:/
expect "password:"
send "$sshPass\r"
expect "*\r"
expect "\r"
