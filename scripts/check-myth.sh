#!/bin/bash
#
# Installs myth if it isn't already installed.
# TODO (thebyrd) install node.js as well
#
echo "--> Checking that Myth is Installed..."
which myth > /dev/null
RESULT=$?
if [ $RESULT -ne 0 ]; then
    echo "--> Installing Myth first."
    sudo npm install -g myth
fi
echo "--> Myth is Installed"
