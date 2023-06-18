#!/bin/bash
export http_proxy="http://192.168.3.1:3129"                                                                                                                                                          
export https_proxy="http://192.168.3.1:3129"
set -x
git config --global http.sslVerify "false"
# git clone https://{USER}:{PASS}@git.santalucia.net/scm/{PROJECTKEY}/{APP_REPO}.git {DESTINATION_DIR}
git clone https://$1:$2@git.santalucia.net/scm/$3/$4.git $5 > /tmp/clone.log 2>&1 