#/bin/bash

cd "$(dirname "$0")"

rm -rf docs
reveal-md course.md --static docs
cp -r images docs 
sed -i 's@/images/@./images/@g' docs/index.html docs/course.html