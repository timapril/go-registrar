# Creating the RDE Bundle

Run the following commands

go run main.go -conf prod -v
./encrypt.sh ./output/ $GPG_PRIV_KEY $ENCRYPT_TARGETS
scp -i ~/.ssh/rdekey ./output/* $RDE_USERNAME@$RDE_UPLOAD_SERVER:

Upload the /output directory contents to the iron mountain RDE server in the
root directory
