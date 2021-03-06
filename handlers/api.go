// Statup
// Copyright (C) 2018.  Hunter Long and the project contributors
// Written by Hunter Long <info@socialeck.com> and the project contributors
//
// https://github.com/hunterlong/statup
//
// The licenses for most software and other practical works are designed
// to take away your freedom to share and change the works.  By contrast,
// the GNU General Public License is intended to guarantee your freedom to
// share and change all versions of a program--to make sure it remains free
// software for all its users.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/hunterlong/statup/core"
	"github.com/hunterlong/statup/core/notifier"
	"github.com/hunterlong/statup/types"
	"github.com/hunterlong/statup/utils"
	"net/http"
	"os"
	"time"
)

type apiResponse struct {
	Status string `json:"status"`
	Object string `json:"type"`
	Id     int64  `json:"id"`
	Method string `json:"method"`
}

func apiIndexHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	var out core.Core
	out = *core.CoreApp
	var services []types.ServiceInterface
	for _, s := range out.Services {
		service := s.Select()
		service.Failures = nil
		services = append(services, core.ReturnService(service))
	}
	out.Services = services
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func apiRenewHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	var err error
	core.CoreApp.ApiKey = utils.NewSHA1Hash(40)
	core.CoreApp.ApiSecret = utils.NewSHA1Hash(40)
	core.CoreApp, err = core.UpdateCore(core.CoreApp)
	if err != nil {
		utils.Log(3, err)
	}
	http.Redirect(w, r, "/settings", http.StatusSeeOther)
}

func apiCheckinHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	checkin := core.SelectCheckin(vars["api"])
	//checkin.Receivehit()
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(checkin)
}

func apiServiceDataHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	service := core.SelectService(utils.StringInt(vars["id"]))
	if service == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	fields := parseGet(r)
	grouping := fields.Get("group")
	startField := utils.StringInt(fields.Get("start"))
	endField := utils.StringInt(fields.Get("end"))

	if startField == 0 || endField == 0 {
		startField = 0
		endField = 99999999999
	}

	obj := core.GraphDataRaw(service, time.Unix(startField, 0).UTC(), time.Unix(endField, 0).UTC(), grouping, "latency")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(obj)
}

func apiServicePingDataHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	service := core.SelectService(utils.StringInt(vars["id"]))
	if service == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	fields := parseGet(r)
	grouping := fields.Get("group")
	startField := utils.StringInt(fields.Get("start"))
	endField := utils.StringInt(fields.Get("end"))
	obj := core.GraphDataRaw(service, time.Unix(startField, 0), time.Unix(endField, 0), grouping, "ping_time")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(obj)
}

func apiServiceHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	service := core.SelectServicer(utils.StringInt(vars["id"]))
	if service == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service.Select())
}

func apiCreateServiceHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	var service *types.Service
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&service)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	newService := core.ReturnService(service)
	_, err = newService.Create(true)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service)
}

func apiServiceUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	service := core.SelectService(utils.StringInt(vars["id"]))
	if service == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	var updatedService *types.Service
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&updatedService)
	updatedService.Id = service.Id
	service = core.ReturnService(updatedService)
	err := service.Update(true)
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	service.Check(true)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service)
}

func apiServiceDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	service := core.SelectService(utils.StringInt(vars["id"]))
	if service == nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	err := service.Delete()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	output := apiResponse{
		Object: "service",
		Method: "delete",
		Id:     service.Id,
		Status: "success",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

func apiAllServicesHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	services := core.Services()
	var servicesOut []*types.Service
	for _, s := range services {
		servicesOut = append(servicesOut, s.Select())
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(servicesOut)
}

func apiUserHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	user, err := core.SelectUser(utils.StringInt(vars["id"]))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func apiUserUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	user, err := core.SelectUser(utils.StringInt(vars["id"]))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	var updateUser *types.User
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&updateUser)
	updateUser.Id = user.Id
	user = core.ReturnUser(updateUser)
	err = user.Update()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func apiUserDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	user, err := core.SelectUser(utils.StringInt(vars["id"]))
	if err != nil {
		http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
		return
	}
	err = user.Delete()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}
	output := apiResponse{
		Object: "user",
		Method: "delete",
		Id:     user.Id,
		Status: "success",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

func apiAllUsersHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	users, err := core.SelectAllUsers()
	if err != nil {
		utils.Log(3, err)
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}

func apiCreateUsersHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	var user *types.User
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	newUser := core.ReturnUser(user)
	uId, err := newUser.Create()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	output := apiResponse{
		Object: "user",
		Method: "create",
		Id:     uId,
		Status: "success",
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

func apiNotifierGetHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	_, notifierObj, err := notifier.SelectNotifier(vars["notifier"])
	if err != nil {
		http.Error(w, fmt.Sprintf("%v notifier was not found", vars["notifier"]), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifierObj)
}

func apiNotifierUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	var notification *notifier.Notification
	fmt.Println(r.Body)
	decoder := json.NewDecoder(r.Body)
	decoder.Decode(&notification)

	notifer, not, err := notifier.SelectNotifier(vars["notifier"])
	if err != nil {
		http.Error(w, fmt.Sprintf("%v notifier was not found", vars["notifier"]), http.StatusInternalServerError)
		return
	}

	notifer.Var1 = notification.Var1
	notifer.Var2 = notification.Var2
	notifer.Host = notification.Host
	notifer.Port = notification.Port
	notifer.Password = notification.Password
	notifer.Username = notification.Username
	notifer.Enabled = notification.Enabled
	notifer.ApiKey = notification.ApiKey
	notifer.ApiSecret = notification.ApiSecret

	_, err = notifier.Update(not, notifer)
	if err != nil {
		utils.Log(3, fmt.Sprintf("issue updating notifier: %v", err))
	}
	notifier.OnSave(notifer.Method)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifer)
}

func apiAllMessagesHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	messages, err := core.SelectMessages()
	if err != nil {
		http.Error(w, fmt.Sprintf("error fetching all messages: %v", err), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(messages)
}

func apiMessageGetHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	message, err := core.SelectMessage(utils.StringInt(vars["id"]))
	if err != nil {
		http.Error(w, fmt.Sprintf("message #%v was not found", vars["id"]), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(message)
}

func apiMessageDeleteHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	message, err := core.SelectMessage(utils.StringInt(vars["id"]))
	if err != nil {
		http.Error(w, fmt.Sprintf("message #%v was not found", vars["id"]), http.StatusInternalServerError)
		return
	}
	err = message.Delete()
	if err != nil {
		http.Error(w, fmt.Sprintf("message #%v could not be deleted %v", vars["id"], err), http.StatusInternalServerError)
		return
	}

	output := apiResponse{
		Object: "message",
		Method: "delete",
		Id:     message.Id,
		Status: "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

func apiMessageUpdateHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	message, err := core.SelectMessage(utils.StringInt(vars["id"]))
	if err != nil {
		http.Error(w, fmt.Sprintf("message #%v was not found", vars["id"]), http.StatusInternalServerError)
		return
	}
	var messageBody *types.Message
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&messageBody)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	messageBody.Id = message.Id
	message = core.ReturnMessage(messageBody)
	_, err = message.Update()
	if err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	output := apiResponse{
		Object: "message",
		Method: "update",
		Id:     message.Id,
		Status: "success",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(output)
}

func apiNotifiersHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	var notifiers []*notifier.Notification
	for _, n := range core.CoreApp.Notifications {
		notif := n.(notifier.Notifier)
		notifiers = append(notifiers, notif.Select())
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifiers)
}

func apiAllServiceFailuresHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	allServices, _ := core.CoreApp.SelectAllServices(false)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allServices)
}

func apiServiceFailuresHandler(w http.ResponseWriter, r *http.Request) {
	if !isAPIAuthorized(r) {
		sendUnauthorizedJson(w, r)
		return
	}
	vars := mux.Vars(r)
	service := core.SelectService(utils.StringInt(vars["id"]))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(service.AllFailures())
}

func sendUnauthorizedJson(w http.ResponseWriter, r *http.Request) {
	data := map[string]interface{}{
		"error": "unauthorized",
		"url":   r.RequestURI,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(data)
}

func isAPIAuthorized(r *http.Request) bool {
	if os.Getenv("GO_ENV") == "test" {
		return true
	}
	if IsAuthenticated(r) {
		return true
	}
	if isAuthorized(r) {
		return true
	}
	return false
}
