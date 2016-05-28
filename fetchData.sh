#!/bin/sh
echo "*** Updating ***"
cd $(dirname $0)
python2 fetchSmashrun.py 
cat tracks.csv | sort | uniq > tracks2.csv
rm tracks.csv
mv tracks2.csv tracks.csv
echo "*** Done ***"

