#!/bin/bash
#
# Starts mongo if it isnt up.
# TODO (thebyrd) install mongod if not avaiable
#
echo "--> Checking that Mongo is Up"
ps -A | grep [m]ongod
RESULT=$?   # returns 0 if mongo eval succeeds
if [ $RESULT -ne 0 ]; then
    echo "--> Starting MongoDB..."
    mongod &
fi
