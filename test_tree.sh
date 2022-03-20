#!/bin/bash

set -eu


cat $0

make build

rm -rf tmp
mkdir -p tmp
mkdir -p tmp/a
echo "x" > tmp/a/x
echo "y" > tmp/a/y
cp -r tmp/a tmp/b
cp -r tmp/a tmp/c


input_path=./tmp
#input_path=$PWD/tmp

list=$(./bin/listfiles ${input_path})
echo "--- list ---"
echo "$list"
echo "--- cleanup manifest ---"
echo "$list" | ./bin/cleanup -l -


# here generate fake manifest to later run cleanup 

cat <<EOF | ./bin/cleanup -m - -t trash | tee tmp_clean.bash
keep	1fb9e3ff04b12d5f	tmp/b/
move	1fb9e3ff04b12d5f	tmp/c/
move	1fb9e3ff04b12d5f	tmp/a/
EOF

bash tmp_clean.bash

# ========================================================
