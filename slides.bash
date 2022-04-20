#/bin/bash

cd "$(dirname "$0")"
NAME=course
rm $NAME
sudo DEBUG=reveal-md reveal-md $NAME.md --puppeteer-launch-args="dumpio:true --no-sandbox --disable-setuid-sandbox" --print $NAME.pdf
sudo chown -R $USER:$USER $NAME.pdf