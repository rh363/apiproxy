
# APIPROXY

This is an api developed for communicate with a nginx server instance for automate the management operation about ip forward on NFS mount service. \
The forward operations run on TCP level, to permise an stream forward and let nfs service work.


## Requirement
This api have some requirement:

 - Nginx installed with stream
 - Nginx configured as described in next section
 - Ports 4444,20048,2049,111 must be open and not used

 
## Configure Nginx

This api work reading nginx configuration files so is very important to respect this guide standard. \
The api and the nginx server configuration files as be tested only on debian OS. \
For configure nginx is possible follow this guide or running the configurenginx.sh script. \
It require the ip address when nginx must listen and the path to apiproxy executable. \
This script only work on ubuntu and debian OS. Must be updated and tested run it on your own. \

To get started you need to install Nginx: 

``` bash
sudo apt-get update
sudo apt-get install nginx nginx-mod-stream
```

Now we can stop nginx by using default site:
``` bash
sudo unlink /etc/nginx/sites-enabled/default
```
Now you can download nginx configuration files avaible in this repo and apply new configurations:
``` bash
sudo mv nginx.conf /etc/nginx/nginx.conf
mkdir /etc/nginx/restart
sudo mv restart.sh /etc/nginx/restart/restart.sh
```
Before apply rproxy.conf you have to modify some line for insert your listening server ip address: ```sudo vi rproxy.conf ```

``` bash
stream {
    include conf/upstreams.config;

    map $remote_addr $upstream2049 {
        include conf/upstream2049.config;
        default     notaccepted;
    }

    map $remote_addr $upstream111 {
        include conf/upstream111.config;
        default     notaccepted;
    }

    map $remote_addr $upstream20048 {
        default     notaccepted;
        include conf/upstream20048.config;
    }

    map $remote_addr $upstream445 {
        include conf/upstream445.config;
        default     notaccepted;
    }

    map $remote_addr $upstream139 {
        default     notaccepted;
        include conf/upstream139.config;
    }

    server {
        listen x.x.x.x:445;    #insert here yout ip
        proxy_pass $upstream445;
    }

    server {
        listen x.x.x.x:139;    #insert here yout ip
        proxy_pass $upstream139;
    }
    
    server {
        listen x.x.x.x:2049;    #insert here yout ip
        proxy_pass $upstream2049;
    }

    server {
        listen x.x.x.x:111;    #insert here yout ip
        proxy_pass $upstream111;
    }

    server {
        listen x.x.x.x:20048;    #insert here yout ip
        proxy_pass $upstream20048;
    }


}
```

Now we can apply it:

``` bash
sudo mv rproxy.conf /etc/nginx/sites-available/rproxy.conf
sudo ln -s /etc/nginx/sites-available/rproxy.conf /etc/nginx/sites-enabled/rproxy.conf
```
To conclude we only have to create some empty file who api populate and use for store forward:

``` bash
sudo mkdir /etc/nginx/conf
sudo mkdir /etc/nginx/record
cd /etc/nginx/conf
sudo touch upstream20048.conf upstream2049.conf upstream111.conf upstream445.conf upstream139.conf upstreams.conf
cd /etc/nginx/record
sudo touch record.txt
```
Upstream20048,2049,111,445,139 are simple files who rproxy import ad use for map specific src address to an nginx upstream. \
scompose it in more file make manage mapped ip more easy by golang code. \
Upstreams file contain all upstreams directive for all port. \
When an client try to connect to this proxy, the proxy try to match the ip address, if it dont match is dropped, if match is redirected to specific upstream contained in upstreams file. \
This is done for ports 2049,20048,111 if client use nfs or for ports 445,139 if client use smb. \
Record.txt contain a simple list of forward with srcIP:destIP:clientname:type format. \
It is usefull for track every forward better and faster. \

Now Nginx server is ready for apiproxy so we can restart it:
``` bash
sudo nginx -t
sudo systemctl restart nginx
```

## Install

For use apiproxy software you can download the apiproxy exec from desired release. \
You can use this by running it or create a new systemd service for automate api start on system boot for example:

```bash
mv apiproxy /usr/local/bin/apiproxy
vi /etc/systemd/system/apiproxy.service
```

Insert:
```bash
[Unit]
Description=apiproxy service

[Service]
ExecStart=/usr/local/bin/apiproxy

[Install]
WantedBy=multi-user.target
```
next:

```bash
systemctl daemon-reload
systemctl enable --now apiproxy.service
```

Now apiproxy tart listening on port 4444 automaticaly.
## How to use

For use this api a client must send http request to the api server.
In Request sections you can see all supported request.
## GET Request


there are six get request:

| request path | request scope |
| --- | --- |
| / | return a list of all supported request in a json file |
| /apiproxy/rproxy/status | return the status of api proxy, it check if "CanRestartNginx" var is true.  <br>if it isn't true user cant modify the nginx config files by api because probably a config file is compromised, sysadmin can mannualy restore it by a backup located in /etc/nginx/record/changes.txt |
| /apiproxy/rproxy/forward | it return all current running forward |
| /apiproxy/rproxy/forward/nfs | it return current running nfs forward |
| /apiproxy/rproxy/forward/smb | it return all current running smb forward |
| /apiproxy/rproxy/conf | it return current config files in json format |



## POST Request

there are two post request:

| request path | body | request scope |
| --- | --- | --- |
| /apiproxy/rproxy/forward/nfs | {  <br>"src" : "\*src address\*",  <br>"dest" : "\*dest address\*",  <br>"client" : "\*client name\*"  <br>} | it create a new nfs stream forward, ir require ip address to redirect, the forward destination and the client name.  <br>the api update the nginx config file for redirect nfs ports (2049,20048,111) and the custom record, next it restart nginx service.  <br>if you use apik3s is recommended use for label "client" the same name of client workspace. |
| /apiproxy/rproxy/forward/smb | {  <br>"src" : "\*src address\*",  <br>"dest" : "\*dest address\*",  <br>"client" : "\*client name\*"  <br>} | it create a new smb stream forward, ir require ip address to redirect, the forward destination and the client name.  <br>the api update the nginx config file for redirect smb ports (445,139) and the custom record, next it restart nginx service.  <br>if you use apik3s is recommended use for label "client" the same name of client workspace. |

## DELETE Request

there are four delete request in this api.

| request path | request scope |
| --- | --- |
| /apiproxy/rproxy/forward/nfs/:client/:src/:dest | it delete an nfs stream forward.  <br>it delete every map source address occurs,and remove this specif forward by the custom record.  <br>if the destination address is never used after changes it remove upstream to this address. |
| /apiproxy/rproxy/forward/smb/:client/:src/:dest | it delete an smb stream forward.  <br>it delete every map source address occurs,and remove this specif forward by the custom record.  <br>if the destination address is never used after changes it remove upstream to this address. |
| /apiproxy/rproxy/forward | it remove all forward and clean all configuration files |

## Used project

- For api build: [gin-gonic](https://github.com/gin-gonic/gin)
- As reverse proxy: [Nginx](https://github.com/nginx) 
## Authors

- [Massaroni Alex](https://www.github.com/rh363)
- [Vona Daniele]()


## Future Updates

- make the api more light
- manage better file compromised failover
