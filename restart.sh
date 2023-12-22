#!/usr/bin/env bash

ROOT_UID=0                                                                                                                                          #root UID
USER_UID=$(id -u)
ERR_NOTROOT=86

if [ "$USER_UID" -ne "$ROOT_UID" ]                                                                                                                  #controlla se l'utente è root
    then
    echo "Must be root to run this function."
    exit $ERR_NOTROOT
    fi
systemctl restart nginx