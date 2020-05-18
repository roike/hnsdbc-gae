#!/bin/bash

CMDNAME=`basename $0`

usage_exit() {
    echo "Usage: $CMDNAME [-t] [-s] [-a] [-f] [-b P|D]" 1>&2
    exit 1
}

while getopts tsafb:h OPT
do
    case $OPT in
        t) FLAG_T="TRUE" ;;
        s) FLAG_S="TRUE" ;;
        a) FLAG_A="TRUE" ;;
        f) FLAG_F="TRUE" ;;
        b) FLAG_B="TRUE" ; VALUE_B=$OPTARG ;;
        h) usage_exit ;;
        \?) usage_exit ;;
    esac
done
shift $((OPTIND - 1))

if [ "$FLAG_T" = "TRUE" ]; then
    echo "Error: no test specified."
fi

if [ "$FLAG_S" = "TRUE" ]; then
    webpack-dev-server --mode=development --config webpack.config.js
fi

if [ "$FLAG_A" = "TRUE" ]; then
    export PROJECT_ID="hnsdbc"
    export DEFAULT_BUCKET=hnsdbc.appspot.com
    go run main/main.go
fi

if [ "$FLAG_F" = "TRUE" ]; then
    prettier-eslint --write 'src/js/*.js'
fi

if [ "$VALUE_B" = "P" ]; then
    webpack --mode production --config webpack.config.js
fi

if [ "$VALUE_B" = "D" ]; then
    webpack --mode development --config webpack.config.js
fi

exit 0
