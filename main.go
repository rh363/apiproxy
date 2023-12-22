package main

/*
NAME=APIPROXY
AUTHOR=RH363
DATE=12/2023
COMPANY=SEEWEB
VERSION=1.0

DESCRIPTION:

THIS IS AN API WRITE IN GO-LANG IT'S POURPOSE IS COMMUNICATE WITH AN NGINX
SERVER RUNNING ON THIS PC FOR MANAGE THE FORWARD CONFIGURATION FILE OF THE NGINX SERVICE.
THIS FILE CAN RUN ONLY IF EVERY COMPONENT IS IN ITS PLACE.
PLEASE READ README DOC BEFORE DEPLOY THIS API.

REQUIREMENT:
	-1: RUN IT AS ROOT
	-2: NGINX INSTALLED AND CONFIGURED HOW DESCRIPTED README.TXT
	-3: CAN USE PORT 2049,20048,111,4444
*/
// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ IMPORTS
import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/gin-gonic/gin"
)

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ NGINX STATE
var CanRestart bool = true

/*
this var is very important, when the api remove old configuration files for write the newer
if this api cant rewrite the new configuration file this var became false and all nginx restart
try is stopped so the old configuration runnning in nginx can still work, an error like this require
sysadmin intervention, if you are a sys admin you can find old configuration in a file : "/etc/nginx/record/changes.txt"
*/
// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ FILES NAME
var ChangesFile string = "changes.txt"
var RestartFile string = "restart.sh"
var NginxConfigFile string = "rproxy.conf"
var RecordFile string = "record.txt"
var UpstreamsFile string = "upstreams.config"
var Upstream20048File string = "upstream20048.config"
var Upstream2049File string = "upstream2049.config"
var Upstream111File string = "upstream111.config"

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ FILES PATH
var ChangesPath string = "/etc/nginx/record/" + ChangesFile
var RestartPath string = "/etc/nginx/restart/" + RestartFile
var NginxConfigPath string = "/etc/nginx/sites-enabled/" + NginxConfigFile
var RecordPath string = "/etc/nginx/record/" + RecordFile
var UpstreamsPath string = "/etc/nginx/conf/" + UpstreamsFile
var Upstream20048Path string = "/etc/nginx/conf/" + Upstream20048File
var Upstream2049Path string = "/etc/nginx/conf/" + Upstream2049File
var Upstream111Path string = "/etc/nginx/conf/" + Upstream111File

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ FILE READ ERROR
var ErrCantReadRecordFile error = errors.New("unable to read " + RecordFile + " located in: " + RecordPath)
var ErrCantReadUpstreamsFile error = errors.New("unable to read " + UpstreamsFile + " located in: " + UpstreamsPath)
var ErrCantReadUpstream20048File error = errors.New("unable to read " + Upstream20048File + " located in: " + Upstream20048Path)
var ErrCantReadUpstream2049File error = errors.New("unable to read " + Upstream2049File + " located in: " + Upstream2049Path)
var ErrCantReadUpstream111File error = errors.New("unable to read " + Upstream111File + " located in: " + Upstream111Path)

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ FILE WRITE ERROR
var ErrCantWriteRecordFile error = errors.New("unable to write " + RecordFile + " located in: " + RecordPath)
var ErrCantWriteUpstreamsFile error = errors.New("unable to write " + UpstreamsFile + " located in: " + UpstreamsPath)
var ErrCantWriteUpstream20048File error = errors.New("unable to write " + Upstream20048File + " located in: " + Upstream20048Path)
var ErrCantWriteUpstream2049File error = errors.New("unable to write " + Upstream2049File + " located in: " + Upstream2049Path)
var ErrCantWriteUpstream111File error = errors.New("unable to write " + Upstream111File + " located in: " + Upstream111Path)

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ NGINX ERROR
var ErrCantRunRestartNginx error = errors.New("unable to restart proxy via api, post method temporarily offline, contact sysadmin please")
var ErrCantUpdateNginx error = errors.New("unable to update nginx with new config, applied changes cleaned")
var ErrCanRestartFalse error = errors.New("CAN RESTART VAR IS ASCUTALLY FALSE YOUR PROXY CONFIG IS ACTUALLY EMPTY IMMEDIATELY CONTACT YOUR SYSADMIN")
var ErrForwardNotFound error = errors.New("forward requested not found")

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ API ERROR
var ErrBadJsonFormat error = errors.New("Json format used is not valid")

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ FILE EDIT FUNCTION
func WriteFile(path string, txt []string) error { // WRITE A NEW FILE IF IT DOESN'T EXIST, ELSE CREATE A NEW FILE, REQUIRE FILE PATH AND CONTENT, RETURN AN ERROR

	File, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	defer File.Close()
	if err != nil {
		fmt.Println(err)
		return err
	}

	writer := bufio.NewWriter(File)

	for _, data := range txt {
		_, err = writer.WriteString(data)
		if err != nil {
			fmt.Println(err)
			return err
		}
	}

	writer.Flush()
	return nil
}

func ReadFile(path string) ([]string, error) { // READ A FILE CONTENT, REQUIRE FILE PATH, RETURN THE CONTENT AND AN ERROR

	File, err := os.Open(path)
	defer File.Close()
	if err != nil {
		return nil, err
	}

	reader := bufio.NewScanner(File)
	reader.Split(bufio.ScanLines)
	var txt []string

	for reader.Scan() {
		if reader.Text() != "" && reader.Text() != " " {
			fmt.Println(reader.Text())
			txt = append(txt, reader.Text())
		}
	}

	return txt, nil
}

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ SCRIPT FUNCTION
func restartnginx() error { // RUN NGINX SCRIPT FOR RESTART NGINX "systemctl restart nginx", RETURN AN ERROR IF USER IS NOT ROOT OR IF CANT RESTART NGINX
	if CanRestart == true { // check if nginx configuration files is not compromised
		cmd := exec.Command("bash", RestartPath)
		if _, err := cmd.Output(); err != nil {
			return err
		}
		return nil
	}
	return ErrCanRestartFalse
}

func removeline(slice []string, index int) []string { // REMOVE A LINE, THIS FUNCTION IS EXPENSIVE BECAUSE IT RECREATE FROM 0 A NEW SLICE BUT IS REQUIRED FOR REMOVE A SLICE MAINTAIN ELEMENT ORDER
	return append(slice[:index], slice[index+1:]...) // it take content in slice until the index passed and append every content after the index to it, ... is used for pass element in second slice one by one
}

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ JSON STRUCT USED

var explain = []Response{
	{Message: "Welcome in my api, in the following messages you can see different option"},
	{Message: "WARNING all instruction behind ** chars must be replaced with your data"},
	{Message: "For get your config file content send a GET request to: /apiproxy/rproxy/conf"},
	{Message: "For get current forward send a GET request to: /apiproxy/rproxy/forward"},
	{Message: "For create a new forward send a POST request to: /apiproxy/rproxy/forward"},
	{Message: `WARNING using POST request for create a new forward require an json file with the following structure:`},
	{Message: `{"src" : "*address to forward*","dest" : "*target address*","client" : "*client name*"}`},
	{Message: `EX: {"src" : "10.2.15.235","dest" : "192.168.1.150","client" : "my client name"}`},
	{Message: `WARNING if you use apik3s is recommended use for label "client" the same name of client workspace`},
	{Message: "For delete an forward send a DELETE request to: /apiproxy/rproxy/forward/*client name*/*address forwarded*/*target address*"},
	{Message: "For get current api status send a GET request to: /apiproxy/rproxy/status"},
}

type ConfigurationFile struct { // configuration file struct for json transfer
	File string `json:"file"`
	Body string `json:"body"`
}

type Response struct { // generic response for json transfer
	Message string `json:"message"`
}

type Forward struct { // forward struct for json transfer
	Src    string `json:"src"`
	Dest   string `json:"dest"`
	Client string `json:"client"`
}

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ API FUNCTIONS

func show(context *gin.Context) {
	context.IndentedJSON(http.StatusOK, explain)
}

func getstatus(context *gin.Context) {
	if CanRestart {
		context.IndentedJSON(http.StatusOK, Response{Message: "all working fine"})
		return
	}
	context.IndentedJSON(http.StatusServiceUnavailable, Response{Message: "CRITICAL ERROR: cant restart nginx service an nginx config file is probably empty please contact sysadmin and restore it by /etc/nginx/record/changes.txt"})
}

func getconfigs(context *gin.Context) { //GET ALL CONFIGURATION FILE CONTENT AND RETURN A JSON FILE
	var configfiles []ConfigurationFile

	var txt string

	upstreamsFile, err := ReadFile(UpstreamsPath) //read upstreams file
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstreamsFile.Error()})
		return
	}

	upstream20048File, err := ReadFile(Upstream20048Path) // read upstream20048 file
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstream20048File.Error()})
		return
	}

	upstream2049File, err := ReadFile(Upstream2049Path) // read upstream2049 file
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstream2049File.Error()})
		return
	}

	upstream111File, err := ReadFile(Upstream111Path) // read upstream111 file
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstream111File.Error()})
		return
	}

	// read every file line for line and append it to []ConfigurationFile called configfiles
	txt = ""
	for _, line := range upstreamsFile {
		txt = txt + line
	}
	configfiles = append(configfiles, ConfigurationFile{File: UpstreamsFile, Body: txt})

	txt = ""
	for _, line := range upstream20048File {
		txt = txt + line
	}
	configfiles = append(configfiles, ConfigurationFile{File: Upstream20048File, Body: txt})

	txt = ""
	for _, line := range upstream2049File {
		txt = txt + line
	}
	configfiles = append(configfiles, ConfigurationFile{File: Upstream2049File, Body: txt})

	txt = ""
	for _, line := range upstream111File {
		txt = txt + line
	}
	configfiles = append(configfiles, ConfigurationFile{File: Upstream111File, Body: txt})

	context.IndentedJSON(http.StatusOK, configfiles) //return json
}

func getforward(context *gin.Context) { // GET ALL FORWARD CONNECTION USING RECORD FILE AN RETURN A JSON OF IT
	var forwards []Forward

	recordFile, err := ReadFile(RecordPath) // read record file
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadRecordFile.Error()})
		return
	}

	for _, line := range recordFile {
		sline := strings.Split(line, ":")                                                     // split record file line by content
		forwards = append(forwards, Forward{Src: sline[0], Dest: sline[1], Client: sline[2]}) // append every content in specific location in []Forward called forwards
	}

	context.IndentedJSON(http.StatusOK, forwards) // return json
}

func createforward(context *gin.Context) { // CREATE A NEW FORWARD, REQUIRE FORWARD DATA IN JSON FORMAT LIKE "Forward" STRUCT

	if err := restartnginx(); err != nil { // check if can run restartnginx script
		if CanRestart == false { //check if configs file is not compromised
			context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCanRestartFalse.Error()})
			return
		}
		context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCantRunRestartNginx.Error()})
		return
	}

	var forward Forward
	if err := context.BindJSON(&forward); err != nil { //get forward data received and bind it to forward var
		context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrBadJsonFormat.Error()})
		return
	}
	upstream20048 := []string{ //create upstream20048 directive
		"\n",
		forward.Src + " " + "svc_" + forward.Client + "_20048;",
	}
	upstream2049 := []string{ //create upstream2049 directive
		"\n",
		forward.Src + " " + "svc_" + forward.Client + "_2049;",
	}
	upstream111 := []string{ //create upstream111 directive
		"\n",
		forward.Src + " " + "svc_" + forward.Client + "_111;",
	}
	upstreams := []string{ //create upstreams directive
		"\n",
		"upstream svc_" + forward.Client + "_2049{",
		"\n",
		"server " + forward.Dest + ":2049;",
		"\n",
		"}",
		"\n",
		"upstream svc_" + forward.Client + "_20048{",
		"\n",
		"server " + forward.Dest + ":20048;",
		"\n",
		"}",
		"\n",
		"upstream svc_" + forward.Client + "_111{",
		"\n",
		"server " + forward.Dest + ":111;",
		"\n",
		"}",
	}
	record := []string{ //create forward record in record file
		"\n",
		forward.Src + ":" + forward.Dest + ":" + forward.Client,
	}

	recordFile, err := ReadFile(RecordPath)
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadRecordFile.Error()})
		return
	}

	for _, line := range recordFile { // check if forward to this Destination already exist
		sline := strings.Split(line, ":")
		if sline[1] == forward.Dest { // if already exist update all config files except for usptreams file
			if err := WriteFile(Upstream20048Path, upstream20048); err != nil {
				context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantWriteUpstream20048File.Error()})
			}
			if err := WriteFile(Upstream2049Path, upstream2049); err != nil {
				context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantWriteUpstream2049File.Error()})
			}
			if err := WriteFile(Upstream111Path, upstream111); err != nil {
				context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantWriteUpstream111File.Error()})
			}
			if err := WriteFile(RecordPath, record); err != nil {
				context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantWriteRecordFile.Error()})
			}
			if err := restartnginx(); err != nil { // restart nginx
				if CanRestart == false {
					context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCanRestartFalse.Error()})
					return
				}
				context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCantUpdateNginx.Error()})
				return
			}
			context.IndentedJSON(http.StatusCreated, forward) //return json requested
			return
		}
	}
	// if doesn't exist update all config files
	if err := WriteFile(Upstream20048Path, upstream20048); err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantWriteUpstream20048File.Error()})
	}
	if err := WriteFile(Upstream2049Path, upstream2049); err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantWriteUpstream2049File.Error()})
	}
	if err := WriteFile(Upstream111Path, upstream111); err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantWriteUpstream111File.Error()})
	}
	if err := WriteFile(UpstreamsPath, upstreams); err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantWriteUpstreamsFile.Error()})
	}
	if err := WriteFile(RecordPath, record); err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantWriteRecordFile.Error()})
	}

	if err := restartnginx(); err != nil { // restart nginx
		if CanRestart == false {
			context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCanRestartFalse.Error()})
			return
		}
		context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCantUpdateNginx.Error()})
		return
	}
	context.IndentedJSON(http.StatusCreated, forward) // return json requested
}

func removeforward(context *gin.Context) { // REMOVE A FORWARD DIRECTIVE, THIS FUNCTION IS A BIT EXPENSIVE IT REQUIRE IN THE URI THIS DATA TO IDENTIFY THE RESOURCE: /:client/:src/:dest

	exist := false

	if err := restartnginx(); err != nil { //check if can restart nginx
		if CanRestart == false { // check if config files is copromise
			context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCanRestartFalse.Error()})
			return
		}
		context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCantRunRestartNginx.Error()})
		return
	}

	// get resource data by uri
	Client := context.Param("client")
	Src := context.Param("src")
	Dest := context.Param("dest")

	// read all config files
	txtupstream20048, err := ReadFile(Upstream20048Path)
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstream20048File.Error()})
		return
	}
	txtupstream2049, err := ReadFile(Upstream2049Path)
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstream2049File.Error()})
		return
	}
	txtupstream111, err := ReadFile(Upstream111Path)
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstream111File.Error()})
		return
	}
	txtupstreams, err := ReadFile(UpstreamsPath)
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstreamsFile.Error()})
		return
	}
	txtrecord, err := ReadFile(RecordPath)
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadRecordFile.Error()})
		return
	}

	// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ EXPLAIN
	/*
				   THIS API USE THE SAME LOGIC FOR ALL CONFIG FILE TO MODIFY
				   THE FOLLOWING SCHEMA EXPLAIN HOW THE FOLLOWING CODE WORK
				   SO YOU MUSTN'T READ USELESS COMMENT LINE:
				   ARE PRESENT TWO ENTITY :
				   -1 FILE (THE ORINAL CONFIGURATION FILE)
				   -2 CHANGES (AN COPY OF ORIGINAL FILE USED FOR STORE CHANGES)

				   IF CLAUSE:

				   ---\ YES
				   ---/

				    ||
				    ||
				    \/
				    NO
				    ---------------------------------------------------------------------------------------------------------------------------------


				   	  	 +------------+     +--------------+     ****************     *********************     +--------------+	  ********************	   +-------------+	   ******************  	  +-----------+	    *****************	  +----------+	   *********************	 +--------------+
				   +-----|FIND FORWARD|-----|REMOVE FORWARD|-----*CHANGES EXIST?*-----*CAN REMOVE CHANGES?*-----|REMOVE CHANGES|------*CAN WRITE CHANGES?*-----|WRITE CHANGES|-----*CAN REMOVE FILE?*-----|REMOVE FILE|-----*CAN WRITE FILE?*-----|WRITE FILE|-----*CAN REMOVE CHANGES?*-----|REMOVE CHANGES|
				   |     +------------+     +--------------+     ****************     *********************     +--------------+	  ********************	   +-------------+     ******************	  +-----------+	    *****************	  +----------+	   *********************	 +--------------+
				   |												|						|												|											|										|										  |
				   |												| 			+-----------+--------------+					+-----------+--------------+				   +--------+--------------+			   +--------+-------+					   +----------+---------------+
			+------+----------+										|			|RETURN CANT REMOVE CHANGES|					|RETURN CANT REMOVE CHANGES|				   |RETURN CANT REMOVE FILE|			   |FILE COMPROMISED|					   |RETURN CANT REMOVE CHANGES|
			|FORWARD NOT EXIST|										|			+--------------------------+					+--------------------------+				   +-----------------------+			   +----------------+					   +--------------------------+
			+-----------------+										|
				   													|
				   													|
				   													|
				   											********************     +-------------+     ******************     +-----------+     ****************      +----------+     *********************     +--------------+
				   											*CAN WRITE CHANGES?*-----|WRITE CHANGES|-----*CAN REMOVE FILE?*-----|REMOVE FILE|-----*CAN WRITE FILE?*-----|WRITE FILE|-----*CAN REMOVE CHANGES?*-----|REMOVE CHANGES|
				   											********************     +-------------+     ******************     +-----------+     ****************      +----------+     *********************     +--------------+
				   													|											|										 |										   |
				   										+-----------+-------------+                  +----------+------------+                   +-------+--------+                    +-----------+--------------+
				   										|RETURN CANT WRITE CHANGES|                  |RETURN CANT REMOVE FILE|				     |FILE COMPROMISED|					   |RETURN CANT REMOVE CHANGES|
				   										+-------------------------+                  +-----------------------+                   +----------------+                    +--------------------------+
		NOTE:
		IF THE API CANT WRITE THE CONFIGURATION FILE (THEORETICALLY IT CAN'T HAPPEN)
		THE API GLOBAL VAR CALLED "CanRestart" BECAME FALSE AND API FUNCTION EXIT
		WITHOUT REMOVE CHANGES FILE SO UNTIL "CanRestart" IS FALSE EVERY RESTART TRY
		OF NGINX SERVICE AND EVERY API FUNCTION WHO MODIFY CONFIGURATION FILE IS DENY, SO
		SYSADMIN CAN RECOVERY LAST CONFIGURATION FILE FROM "/etc/nginx/record/changes.txt"
		AND RESTORE IT MANNUALY.
	*/

	// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ RECORD FILE MODIFY

	for i, line := range txtrecord {
		if line == Src+":"+Dest+":"+Client {
			exist = true

			fmt.Println("forward found in: record")

			txtrecord = removeline(txtrecord, i)
			for i := range txtrecord {
				if i < len(txtrecord)-1 {
					txtrecord[i] = txtrecord[i] + "\n"
				}
			}

			if _, err := os.Stat(ChangesPath); err != nil {
				fmt.Println("changes file not found")
				if err := WriteFile(ChangesPath, txtrecord); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file created")
				if err := os.Remove(RecordPath); err != nil {
					fmt.Println("cant remove record file")
					return
				}
				fmt.Println("record file removed")
				if err := WriteFile(RecordPath, txtrecord); err != nil {
					fmt.Println("cant write record file")
					CanRestart = false
					return
				}
				fmt.Println("record file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
			} else {
				fmt.Println("changes file found")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
				if err := WriteFile(ChangesPath, txtrecord); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file writed")
				if err := os.Remove(RecordPath); err != nil {
					fmt.Println("cant remove record file")
					if err := os.Remove(ChangesPath); err != nil {
						fmt.Println("cant remove changes file")
						return
					}
					fmt.Println("changes file removes")
					return
				}
				fmt.Println("record file removed")
				if err := WriteFile(RecordPath, txtrecord); err != nil {
					fmt.Println("cant write record file")
					CanRestart = false
					return
				}
				fmt.Println("record file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
				}
				fmt.Println("changes file removed")
			}
		}
	}

	if exist == false {
		context.IndentedJSON(http.StatusNotFound, Response{Message: ErrForwardNotFound.Error()})
		return
	}
	// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ UPSTREAM111 FILE MODIFY
	for i, line := range txtupstream111 {
		if line == Src+" "+"svc_"+Client+"_111;" {
			fmt.Println("forward found in: upstream111")

			txtupstream111 = removeline(txtupstream111, i)
			for i := range txtupstream111 {
				if i < len(txtupstream111)-1 {
					txtupstream111[i] = txtupstream111[i] + "\n"
				}
			}

			if _, err := os.Stat(ChangesPath); err != nil {
				fmt.Println("changes file not found")
				if err := WriteFile(ChangesPath, txtupstream111); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file created")
				if err := os.Remove(Upstream111Path); err != nil {
					fmt.Println("cant remove Upstream111 file")
					return
				}
				fmt.Println("Upstream111 file removed")
				if err := WriteFile(Upstream111Path, txtupstream111); err != nil {
					fmt.Println("cant write upstream111 file")
					CanRestart = false
					return
				}
				fmt.Println("Upstream111 file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
			} else {
				fmt.Println("changes file found")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
				if err := WriteFile(ChangesPath, txtupstream111); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file writed")
				if err := os.Remove(Upstream111Path); err != nil {
					fmt.Println("cant remove Upstream111 file")
					if err := os.Remove(ChangesPath); err != nil {
						fmt.Println("cant remove changes file")
						return
					}
					fmt.Println("changes file removes")
					return
				}
				fmt.Println("Upstream111 file removed")
				if err := WriteFile(Upstream111Path, txtupstream111); err != nil {
					fmt.Println("cant write Upstream111 file")
					CanRestart = false
					return
				}
				fmt.Println("Upstream111 file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
				}
				fmt.Println("changes file removed")
			}
		}
	}
	// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ UPSTREAM20048 FILE MODIFY
	for i, line := range txtupstream20048 {
		if line == Src+" "+"svc_"+Client+"_20048;" {
			fmt.Println("forward found in: upstream20048")

			txtupstream20048 = removeline(txtupstream20048, i)
			for i := range txtupstream20048 {
				if i < len(txtupstream20048)-1 {
					txtupstream20048[i] = txtupstream20048[i] + "\n"
				}
			}

			if _, err := os.Stat(ChangesPath); err != nil {
				fmt.Println("changes file not found")
				if err := WriteFile(ChangesPath, txtupstream20048); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file created")
				if err := os.Remove(Upstream20048Path); err != nil {
					fmt.Println("cant remove Upstream20048 file")
					return
				}
				fmt.Println("Upstream20048 file removed")
				if err := WriteFile(Upstream20048Path, txtupstream20048); err != nil {
					fmt.Println("cant write upstream20048 file")
					CanRestart = false
					return
				}
				fmt.Println("Upstream20048 file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
			} else {
				fmt.Println("changes file found")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
				if err := WriteFile(ChangesPath, txtupstream20048); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file writed")
				if err := os.Remove(Upstream20048Path); err != nil {
					fmt.Println("cant remove Upstream20048 file")
					if err := os.Remove(ChangesPath); err != nil {
						fmt.Println("cant remove changes file")
						return
					}
					fmt.Println("changes file removes")
					return
				}
				fmt.Println("Upstream20048 file removed")
				if err := WriteFile(Upstream20048Path, txtupstream20048); err != nil {
					fmt.Println("cant write Upstream20048 file")
					CanRestart = false
					return
				}
				fmt.Println("Upstream20048 file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
				}
				fmt.Println("changes file removed")
			}
		}
	}

	// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ UPSTREAM2049 FILE MODIFY
	for i, line := range txtupstream2049 {
		if line == Src+" "+"svc_"+Client+"_2049;" {
			fmt.Println("forward found in: upstream2049")

			txtupstream2049 = removeline(txtupstream2049, i)
			for i := range txtupstream2049 {
				if i < len(txtupstream2049)-1 {
					txtupstream2049[i] = txtupstream2049[i] + "\n"
				}
			}

			if _, err := os.Stat(ChangesPath); err != nil {
				fmt.Println("changes file not found")
				if err := WriteFile(ChangesPath, txtupstream2049); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file created")
				if err := os.Remove(Upstream2049Path); err != nil {
					fmt.Println("cant remove Upstream2049 file")
					return
				}
				fmt.Println("Upstream2049 file removed")
				if err := WriteFile(Upstream2049Path, txtupstream2049); err != nil {
					fmt.Println("cant write txt.txt")
					CanRestart = false
					return
				}
				fmt.Println("Upstream2049 file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
			} else {
				fmt.Println("changes file found")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
				if err := WriteFile(ChangesPath, txtupstream2049); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file writed")
				if err := os.Remove(Upstream2049Path); err != nil {
					fmt.Println("cant remove Upstream2049 file")
					if err := os.Remove(ChangesPath); err != nil {
						fmt.Println("cant remove changes file")
						return
					}
					fmt.Println("changes file removes")
					return
				}
				fmt.Println("Upstream2049 file removed")
				if err := WriteFile(Upstream2049Path, txtupstream2049); err != nil {
					fmt.Println("cant write Upstream2049 file")
					CanRestart = false
					return
				}
				fmt.Println("Upstream2049 file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
				}
				fmt.Println("changes file removed")
			}
		}
	}
	// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ UPSTREAMS FILE MODIFY
	recordFile, err := ReadFile(RecordPath) // read new record file
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadRecordFile.Error()})
		return
	}

	for _, line := range recordFile { //check new record file line for line
		sline := strings.Split(line, ":")
		if sline[1] == Dest { // if in the new record file the destination ip address to remove is used by another forward restart nginx without modify upstreams file
			if err := restartnginx(); err != nil {
				if CanRestart == false {
					context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCanRestartFalse.Error()})
					return
				}
				context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCantUpdateNginx.Error()})
				return
			}
			context.IndentedJSON(http.StatusOK, Forward{Src: Src, Dest: Dest, Client: Client})
			return
		} // destination ip address is no more used api can modify upstreams file
	}
	// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ UPSTREAMS PORT 111 CLAUSE
	for i, line := range txtupstreams {
		if line == "upstream svc_"+Client+"_111{" && txtupstreams[i+1] == "server "+Dest+":111;" {
			fmt.Println("forward found in upstreams")

			txtupstreams = removeline(txtupstreams, i)
			txtupstreams = removeline(txtupstreams, i)
			txtupstreams = removeline(txtupstreams, i)

			for i := range txtupstreams {
				if i < len(txtupstreams)-1 {
					txtupstreams[i] = txtupstreams[i] + "\n"
				}
			}

			if _, err := os.Stat(ChangesPath); err != nil {
				fmt.Println("changes file not found")
				if err := WriteFile(ChangesPath, txtupstreams); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file created")
				if err := os.Remove(UpstreamsPath); err != nil {
					fmt.Println("cant remove Upstreams file")
					return
				}
				fmt.Println("Upstreams file removed")
				if err := WriteFile(UpstreamsPath, txtupstreams); err != nil {
					fmt.Println("cant write txt.txt")
					CanRestart = false
					return
				}
				fmt.Println("Upstreams file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
			} else {
				fmt.Println("changes file found")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
				if err := WriteFile(ChangesPath, txtupstreams); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file writed")
				if err := os.Remove(UpstreamsPath); err != nil {
					fmt.Println("cant remove Upstreams file")
					if err := os.Remove(ChangesPath); err != nil {
						fmt.Println("cant remove changes file")
						return
					}
					fmt.Println("changes file removes")
					return
				}
				fmt.Println("Upstreams file removed")
				if err := WriteFile(UpstreamsPath, txtupstreams); err != nil {
					fmt.Println("cant write Upstreams file")
					CanRestart = false
					return
				}
				fmt.Println("Upstreams file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
				}
				fmt.Println("changes file removed")
			}
		}

	}
	// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ UPSTREAMS PORT 2049 CLAUSE
	txtupstreams, err = ReadFile(UpstreamsPath)
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstreamsFile.Error()})
		return
	}

	for i, line := range txtupstreams {
		if line == "upstream svc_"+Client+"_2049{" && txtupstreams[i+1] == "server "+Dest+":2049;" {
			fmt.Println("forward found in upstreams")

			txtupstreams = removeline(txtupstreams, i)
			txtupstreams = removeline(txtupstreams, i)
			txtupstreams = removeline(txtupstreams, i)

			for i := range txtupstreams {
				if i < len(txtupstreams)-1 {
					txtupstreams[i] = txtupstreams[i] + "\n"
				}
			}

			if _, err := os.Stat(ChangesPath); err != nil {
				fmt.Println("changes file not found")
				if err := WriteFile(ChangesPath, txtupstreams); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file created")
				if err := os.Remove(UpstreamsPath); err != nil {
					fmt.Println("cant remove Upstreams file")
					return
				}
				fmt.Println("Upstreams file removed")
				if err := WriteFile(UpstreamsPath, txtupstreams); err != nil {
					fmt.Println("cant write txt.txt")
					CanRestart = false
					return
				}
				fmt.Println("Upstreams file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
			} else {
				fmt.Println("changes file found")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
				if err := WriteFile(ChangesPath, txtupstreams); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file writed")
				if err := os.Remove(UpstreamsPath); err != nil {
					fmt.Println("cant remove Upstreams file")
					if err := os.Remove(ChangesPath); err != nil {
						fmt.Println("cant remove changes file")
						return
					}
					fmt.Println("changes file removes")
					return
				}
				fmt.Println("Upstreams file removed")
				if err := WriteFile(UpstreamsPath, txtupstreams); err != nil {
					fmt.Println("cant write Upstreams file")
					CanRestart = false
					return
				}
				fmt.Println("Upstreams file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
				}
				fmt.Println("changes file removed")
			}
		}

	}
	// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ UPSTREAMS PORT 20048 CLAUSE
	txtupstreams, err = ReadFile(UpstreamsPath)
	if err != nil {
		context.IndentedJSON(http.StatusInternalServerError, Response{Message: ErrCantReadUpstreamsFile.Error()})
		return
	}

	for i, line := range txtupstreams {
		if line == "upstream svc_"+Client+"_20048{" && txtupstreams[i+1] == "server "+Dest+":20048;" {
			fmt.Println("forward found in upstreams")

			txtupstreams = removeline(txtupstreams, i)
			txtupstreams = removeline(txtupstreams, i)
			txtupstreams = removeline(txtupstreams, i)

			for i := range txtupstreams {
				if i < len(txtupstreams)-1 {
					txtupstreams[i] = txtupstreams[i] + "\n"
				}
			}

			if _, err := os.Stat(ChangesPath); err != nil {
				fmt.Println("changes file not found")
				if err := WriteFile(ChangesPath, txtupstreams); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file created")
				if err := os.Remove(UpstreamsPath); err != nil {
					fmt.Println("cant remove Upstreams file")
					return
				}
				fmt.Println("Upstreams file removed")
				if err := WriteFile(UpstreamsPath, txtupstreams); err != nil {
					fmt.Println("cant write txt.txt")
					CanRestart = false
					return
				}
				fmt.Println("Upstreams file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
			} else {
				fmt.Println("changes file found")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
					return
				}
				fmt.Println("changes file removed")
				if err := WriteFile(ChangesPath, txtupstreams); err != nil {
					fmt.Println("cant write changes file")
					return
				}
				fmt.Println("changes file writed")
				if err := os.Remove(UpstreamsPath); err != nil {
					fmt.Println("cant remove Upstreams file")
					if err := os.Remove(ChangesPath); err != nil {
						fmt.Println("cant remove changes file")
						return
					}
					fmt.Println("changes file removes")
					return
				}
				fmt.Println("Upstreams file removed")
				if err := WriteFile(UpstreamsPath, txtupstreams); err != nil {
					fmt.Println("cant write Upstreams file")
					CanRestart = false
					return
				}
				fmt.Println("Upstreams file writed")
				if err := os.Remove(ChangesPath); err != nil {
					fmt.Println("cant remove changes file")
				}
				fmt.Println("changes file removed")
			}
		}

	}
	if err := restartnginx(); err != nil { // restart nginx
		if CanRestart == false {
			context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCanRestartFalse.Error()})
			return
		}
		context.IndentedJSON(http.StatusBadRequest, Response{Message: ErrCantUpdateNginx.Error()})
		return
	}
	context.IndentedJSON(http.StatusOK, Forward{Src: Src, Dest: Dest, Client: Client}) // return json
}

// ++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++++ MAIN FUNCTION

func main() {

	if os.Geteuid() != 0 {
		log.Panicln("ERROR: USER MUST BE ROOT TO RUN THIS SCRIPT")
	}
	if _, err := os.Stat(Upstream111Path); err != nil {
		log.Panicln("ERROR: \"" + Upstream111File + "\" NOT PRESENT IN: \"" + Upstream111Path + "\"")
	}
	if _, err := os.Stat(Upstream2049Path); err != nil {
		log.Panicln("ERROR: \"" + Upstream2049File + "\" NOT PRESENT IN: \"" + Upstream2049Path + "\"")
	}
	if _, err := os.Stat(Upstream20048Path); err != nil {
		log.Panicln("ERROR: \"" + Upstream20048File + "\" NOT PRESENT IN: \"" + Upstream20048Path + "\"")
	}
	if _, err := os.Stat(UpstreamsPath); err != nil {
		log.Panicln("ERROR: \"" + UpstreamsPath + "\" NOT PRESENT IN: \"" + UpstreamsPath + "\"")
	}
	if _, err := os.Stat(RestartPath); err != nil {
		log.Panicln("ERROR: \"" + RestartFile + "\" NOT PRESENT IN: \"" + RestartPath + "\"")
	}

	if _, err := os.Stat(NginxConfigPath); err != nil {
		log.Panicln("ERROR: \"" + NginxConfigFile + "\" NOT PRESENT IN: \"" + NginxConfigPath + "\"")
	}

	router := gin.Default()
	router.GET("/", show)
	router.GET("/apiproxy/rproxy/conf", getconfigs)
	router.GET("/apiproxy/rproxy/status", getstatus)
	router.GET("/apiproxy/rproxy/forward", getforward)
	router.POST("/apiproxy/rproxy/forward", createforward)
	router.DELETE("/apiproxy/rproxy/forward/:client/:src/:dest", removeforward)
	router.Run("0.0.0.0:4444")

}
