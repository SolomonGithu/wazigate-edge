package api

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/Waziup/wazigate-edge/clouds"
	"github.com/Waziup/wazigate-edge/tools"
	"github.com/globalsign/mgo/bson"

	routing "github.com/julienschmidt/httprouter"
)

// GetClouds implements GET /clouds
func GetClouds(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	data, err := json.Marshal(clouds.GetClouds())
	if err != nil {
		log.Printf("[ERR  ] Error %v", err)
		http.Error(resp, "internal server error", http.StatusInternalServerError)
		return
	}
	resp.Header().Set("Content-Type", "application/json")
	resp.Write(data)
}

// PostClouds implements POST /clouds
func PostClouds(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloud := &clouds.Cloud{}
	decoder := json.NewDecoder(req.Body)

	if err := decoder.Decode(cloud); err != nil {
		http.Error(resp, "bad Request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if cloud.ID == "" {
		cloud.ID = bson.NewObjectId().Hex()
	}

	if _, err := url.Parse(cloud.REST); err != nil {
		http.Error(resp, "bad request: mal formatted REST address", http.StatusBadRequest)
		return
	}

	if cloud.MQTT != "" {
		if _, err := url.Parse(cloud.MQTT); err != nil {
			http.Error(resp, "bad request: mal formatted MQTT address", http.StatusBadRequest)
			return
		}
	}

	if err := clouds.AddCloud(cloud); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("[CLOUD] Created %q.", cloud.ID)

	writeCloudFile()
	resp.Write([]byte(cloud.ID))
}

// DeleteCloud implements DELETE /clouds/{cloudID}
func DeleteCloud(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	if !clouds.RemoveCloud(cloudID) {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	log.Printf("[CLOUD] Deleted.")
}

// GetCloud implements GET /clouds/{cloudID}
func GetCloud(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	resp.Header().Set("Content-Type", "application/json")
	data, _ := json.Marshal(cloud)
	resp.Write(data)
}

// PostCloudRESTAddr implements POST /clouds/{cloudID}/rest
func PostCloudRESTAddr(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	if !cloud.Paused || cloud.Pausing {
		http.Error(resp, "bad request: cloud is paused or pausing", http.StatusBadRequest)
		return
	}

	var addr string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&addr); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if _, err := url.Parse(addr); err != nil {
		http.Error(resp, "bad request: mal formatted address", http.StatusBadRequest)
		return
	}

	cloud.REST = addr
	log.Printf("[CLOUD] Changed REST addr %q", cloud.REST)
	writeCloudFile()
}

// PostCloudMQTTAddr implements POST /clouds/{cloudID}/name
func PostCloudName(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	body, err := tools.ReadAll(req.Body)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	var name string
	contentType := req.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "application/json") {
		err = json.Unmarshal(body, &name)
		if err != nil {
			http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else {
		name = string(body)
	}

	cloud.Name = name
	log.Printf("[CLOUD] Changed name %q", name)
	writeCloudFile()
}

// PostCloudMQTTAddr implements POST /clouds/{cloudID}/mqtt
func PostCloudMQTTAddr(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	if !cloud.Paused || cloud.Pausing {
		http.Error(resp, "bad request: cloud is paused or pausing", http.StatusBadRequest)
		return
	}

	var addr string
	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&addr); err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	if addr != "" {
		if _, err := url.Parse(addr); err != nil {
			http.Error(resp, "bad request: mal formatted address", http.StatusBadRequest)
			return
		}
	}

	cloud.MQTT = addr
	log.Printf("[CLOUD] Changed MQTT addr %q", cloud.MQTT)
	writeCloudFile()
}

// PostCloudUsername implements POST /clouds/{cloudID}/username
func PostCloudUsername(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(req.Body)
	var username string
	err := decoder.Decode(&username)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	status, err := cloud.SetUsername(username)
	if err != nil {
		http.Error(resp, err.Error(), status)
		return
	}
	resp.WriteHeader(status)
	writeCloudFile()
}

// PostCloudToken implements POST /clouds/{cloudID}/token
func PostCloudToken(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	decoder := json.NewDecoder(req.Body)
	var token string
	err := decoder.Decode(&token)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}
	status, err := cloud.SetToken(token)
	if err != nil {
		http.Error(resp, err.Error(), status)
		return
	}
	resp.WriteHeader(status)
	writeCloudFile()
}

// PostCloudPaused implements POST /clouds/{cloudID}/paused
func PostCloudPaused(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	var paused bool
	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&paused)
	if err != nil {
		http.Error(resp, "bad request: "+err.Error(), http.StatusBadRequest)
		return
	}

	status, err := cloud.SetPaused(paused)
	resp.WriteHeader(status)
	if err != nil {
		resp.Write([]byte(err.Error()))
	} else {
		if paused {
			log.Printf("[CLOUD] Paused synchronization.")
		} else {
			log.Printf("[CLOUD] Resumed synchronization.")
		}
	}

	writeCloudFile()
}

// GetCloudStatus implements GET /clouds/{cloudID}/status
func GetCloudStatus(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	type S struct {
		Entity *clouds.Entity `json:"entity"`
		Status *clouds.Status `json:"status"`
	}

	resp.Header().Set("Content-Type", "application/json; charset=utf-8")

	resp.Write([]byte{'['})

	cloud.StatusMutex.Lock()
	count := 0
	for entity, status := range cloud.Status {
		if count != 0 {
			resp.Write([]byte{','})
		}
		data, _ := json.Marshal(S{&entity, status})
		resp.Write(data)
		count++
	}

	cloud.StatusMutex.Unlock()

	resp.Write([]byte{']'})
}

// GetCloudEvents implements GET /clouds/{cloudID}/events
func GetCloudEvents(resp http.ResponseWriter, req *http.Request, params routing.Params) {

	cloudID := params.ByName("cloud_id")
	cloud := clouds.GetCloud(cloudID)
	if cloud == nil {
		http.Error(resp, "not found: no cloud with that id", http.StatusNotFound)
		return
	}

	resp.Header().Set("Content-Type", "application/json; charset=utf-8")

	cloud.StatusMutex.Lock()
	data, _ := json.Marshal(cloud.Events)
	resp.Write(data)
	cloud.StatusMutex.Unlock()
}

////////////////////////////////////////////////////////////////////////////////

func getCloudsFile() string {
	cloudsFile := os.Getenv("WAZIUP_CLOUDS_FILE")
	if cloudsFile == "" {
		return "clouds.json"
	}
	return cloudsFile
}

func writeCloudFile() {
	cloudsFile := getCloudsFile()
	file, err := os.Create(cloudsFile)
	if err != nil {
		log.Printf("[Err  ] Can not read %q: %s", cloudsFile, err.Error())
		return
	}
	defer file.Close()
	err = clouds.WriteCloudConfig(file)
	if err != nil {
		log.Printf("[Err  ] Can not read %q: %s", cloudsFile, err.Error())
	}
}
