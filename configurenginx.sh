#!/usr/bin/env bash

ROOT_UID=0                                                                                                                                          #root UID
USER_UID=$(id -u)
ERR_NOTROOT=86

wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xvf go1.21.5.linux-amd64.tar.gz
echo "export PATH=\$PATH:/usr/local/go/bin">>~/.profile
sopurce ~/.profile

if [ "$USER_UID" -ne "$ROOT_UID" ]                                                                                                                  #controlla se l'utente è root
    then
    echo "Must be root to run this function."
    exit $ERR_NOTROOT
    fi
echo "<<SYSTEM UPGRADE>>"
apt-get update
apt-get upgrade -y
echo "<<INSTALL NGINX>>"
apt-get install nginx -y 
apt-get install nginx-mod-stream -y
echo "<<REMOVE DEFAULT NGINX CONFIGURATION>>"
unlink /etc/nginx/sites-enabled/default

echo "
user www-data;
worker_processes auto;
pid /run/nginx.pid;
include /etc/nginx/modules-enabled/*.conf;

events {
	worker_connections 768;
	# multi_accept on;
}

include /etc/nginx/conf.d/*.conf;
include /etc/nginx/sites-enabled/*;
" > /etc/nginx/nginx.conf

echo "<<NGINX.CONF READY>>"

echo "
stream {
    include conf/upstreams.config;

    map \$remote_addr \$upstream2049 {
        include conf/upstream2049.config;
        default     notaccepted;
    }

    map \$remote_addr \$upstream111 {
        include conf/upstream111.config;
        default     notaccepted;
    }

    map \$remote_addr \$upstream20048 {
        default     notaccepted;
        include conf/upstream20048.config;
    }

    map \$remote_addr \$upstream445 {
        include conf/upstream445.config;
        default     notaccepted;
    }

    map \$remote_addr \$upstream139 {
        default     notaccepted;
        include conf/upstream139.config;
    }

    server {
        listen $1:445;    
        proxy_pass \$upstream445;
    }

    server {
        listen $1:139;    
        proxy_pass \$upstream139;
    }
    
    server {
        listen $1:2049;    
        proxy_pass \$upstream2049;
    }

    server {
        listen $1:111;    
        proxy_pass \$upstream111;
    }

    server {
        listen $1:20048;    
        proxy_pass \$upstream20048;
    }

}
" > /etc/nginx/sites-available/rproxy.conf
echo "<<RPROXY.CONF READY>>"


mkdir /etc/nginx/restart
echo "
#!/usr/bin/env bash

ROOT_UID=0
USER_UID=$(id -u)
ERR_NOTROOT=86

if [ "\$USER_UID" -ne "\$ROOT_UID" ]
    then
    echo "Must be root to run this function."
    exit \$ERR_NOTROOT
    fi
systemctl restart nginx

" > /etc/nginx/restart/restart.sh
echo "<<RESTART.SH READY>>"

ln -s /etc/nginx/sites-available/rproxy.conf /etc/nginx/sites-enabled/rproxy.conf
echo "<<RPROXY.CONF ENABLED>>"

mkdir /etc/nginx/conf
mkdir /etc/nginx/record
touch /etc/nginx/conf/upstream20048.config /etc/nginx/conf/upstream2049.config /etc/nginx/conf/upstream111.config /etc/nginx/conf/upstream445.config /etc/nginx/conf/upstream139.config /etc/nginx/conf/upstreams.config
echo "<<UPSTREAMS FILES READY>>"
touch /etc/nginx/record/record.txt
echo "<<RECORD FILE READY>>"


sudo nginx -t
sudo systemctl restart nginx
echo "<<NGINX CONFIGURATION COMPLETE>>"


mv $2 /usr/local/bin/apiproxy
echo "
[Unit]
Description=apiproxy service

[Service]
ExecStart=/usr/local/bin/apiproxy

[Install]
WantedBy=multi-user.target
" > /etc/systemd/system/apiproxy.service
echo "<<API PROXY SERVICE CONFIGURED>>"

systemctl daemon-reload
systemctl enable --now apiproxy.service
echo "<<API PROXY SERVICE STARTED>>"
