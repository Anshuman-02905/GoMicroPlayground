package main

import (
	"broker/event"
	"broker/logs"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/rpc"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AuthPayload struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}
type LogPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

type MailPayLoad struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Subject string `json:"subject"`
	Message string `json:"message"`
}

type RequestPayload struct {
	Action string      `json:"action"`
	Auth   AuthPayload `json:"auth,omitempty"`
	Log    LogPayload  `json:"log,omitempty"`
	Mail   MailPayLoad `json:"mail,omitempty"`
}

func (app *Config) Broker(w http.ResponseWriter, r *http.Request) {

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Hit the broker",
	}

	_ = app.WriteJson(w, http.StatusOK, payload, r)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)

}
func (app *Config) HandleSubmission(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload
	err := app.readJson(w, r, &requestPayload)

	if err != nil {
		app.errorJson(w, r, err)

		return
	}
	switch requestPayload.Action {
	case "auth":

		app.authenticate(w, requestPayload.Auth)
	case "log":
		//app.logitem(w, r, requestPayload.Log)
		//app.logeventViaRabbot(w, r, requestPayload.Log)
		app.logItemViaRPC(w, r, requestPayload.Log)

	case "mail":
		app.sendMail(w, r, requestPayload.Mail)

	default:
		app.errorJson(w, r, errors.New("Unknown Action"))
	}
}

func (app *Config) authenticate(w http.ResponseWriter, a AuthPayload) {
	//Create some json we will send to the auth microservice
	jsonData, _ := json.MarshalIndent(a, "", "\t")
	//call the service
	request, err := http.NewRequest("POST", "http://authentication-service/authenticate", bytes.NewBuffer(jsonData))

	if err != nil {
		app.errorJson(w, request, err)
		log.Println("Error Creating a new Request in Authenticate Hanlder", err)
		return
	}
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJson(w, request, err)
		log.Println("Error Posting Request in Authenticate Hanlder", err)

		return
	}
	defer response.Body.Close()

	//make sure we  get the correct status
	if response.StatusCode == http.StatusUnauthorized {
		app.errorJson(w, request, errors.New("Invalid Credentials"))
		log.Println("Error Invalid Credentials STATUS UNATHORIZED")
		return

	} else if response.StatusCode != http.StatusAccepted {
		app.errorJson(w, request, errors.New("Error Calling Auth Service"))
		log.Println("Error Calling Auth Service")

		return
	}
	//create a variable  we'll read respinse.Body into

	var jsonFromService jsonResponse

	//decodet the json from the authservice

	err = json.NewDecoder(response.Body).Decode(&jsonFromService)

	if err != nil {
		app.errorJson(w, request, err)
		return
	}

	if jsonFromService.Error {
		app.errorJson(w, request, err, http.StatusUnauthorized)
		return
	}
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Authenticated!"
	payload.Data = jsonFromService.Data

	app.WriteJson(w, http.StatusAccepted, payload, request)

}

func (app *Config) logitem(w http.ResponseWriter, r *http.Request, entry LogPayload) {
	// create some json we will sent to the logger-service
	jsonData, _ := json.MarshalIndent(entry, "", "\t")
	// call the service
	logServiceUrl := "http://logger-service/log"
	// request, err := http.NewRequest("POST", "http://logger-service/", bytes.NewBuffer(jsonData))
	request, err := http.NewRequest("POST", logServiceUrl, bytes.NewBuffer(jsonData))

	if err != nil {
		app.errorJson(w, r, err)
		log.Println("Error creating a new Request in Logger Handler", err)
		return
	}
	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJson(w, r, err)
		log.Println("Error Posting Request in Authenticate handler", err)
	}
	defer response.Body.Close()

	// make sure we get the correct status
	if response.StatusCode != http.StatusAccepted {
		app.errorJson(w, request, err, response.StatusCode)
		log.Println("Error getting the correct status code post calling the service")
		return
	}
	// Decode the json from logger service
	var jsonFromService jsonResponse

	err = json.NewDecoder(response.Body).Decode(&jsonFromService)
	if err != nil {
		app.errorJson(w, r, err)
		log.Println("Error Decoding the response")
		return
	}
	if jsonFromService.Error {
		app.errorJson(w, r, err)
		log.Println("Error message in the respnse from service")
		return
	}
	// send response
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged"

	app.WriteJson(w, http.StatusAccepted, payload, r)

}

func (app *Config) sendMail(w http.ResponseWriter, r *http.Request, msg MailPayLoad) {
	jsonData, _ := json.MarshalIndent(msg, "", "\t")

	//call the mail service
	mailServiceURL := "http://mailer-service/send"

	//post to the mail service
	request, err := http.NewRequest("POST", mailServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		app.errorJson(w, r, err)
		return
	}
	request.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		app.errorJson(w, r, err)
		return
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusAccepted {
		app.errorJson(w, r, errors.New("error Calling mail service"))
		return
	}
	//send back response
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Message sent to " + msg.To

	app.WriteJson(w, http.StatusAccepted, payload, r)

}

func (app *Config) logeventViaRabbot(w http.ResponseWriter, r *http.Request, entry LogPayload) {
	err := app.pushToQueue(entry.Name, entry.Data)
	if err != nil {
		app.errorJson(w, r, err)
		return
	}
	var payload jsonResponse
	payload.Error = false
	payload.Message = "Logged via rabbit MQ"

	app.WriteJson(w, http.StatusAccepted, payload, r)

}
func (app *Config) pushToQueue(name, msg string) error {
	emitter, err := event.NewEventEmitter(app.Rabbit)
	if err != nil {
		return err
	}
	payload := LogPayload{
		Name: name,
		Data: msg,
	}
	j, _ := json.MarshalIndent(&payload, "", "\t")
	err = emitter.Push(string(j), "log.INFO")
	if err != nil {
		return err
	}
	return nil
}

type RPCPayload struct {
	Name string
	Data string
}

func (app *Config) logItemViaRPC(w http.ResponseWriter, r *http.Request, l LogPayload) {
	client, err := rpc.Dial("tcp", "logger-service:5000")
	if err != nil {
		log.Println("Error  at Dialing TCP")

		app.errorJson(w, r, err)
		return
	}

	RPCPayload := RPCPayload{
		Name: l.Name,
		Data: l.Data,
	}
	var result string
	err = client.Call("RPCServer.LogInfo", RPCPayload, &result)
	if err != nil {
		log.Println("Error  at calling client", err)
		app.errorJson(w, r, err)
		return
	}

	payload := jsonResponse{
		Error:   false,
		Message: result,
	}
	app.WriteJson(w, http.StatusAccepted, payload, r)
}

func (app *Config) LogviaGRPC(w http.ResponseWriter, r *http.Request) {
	var requestPayload RequestPayload

	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		log.Println("eror Reading requuest Payload")
		app.errorJson(w, r, err)
		return
	}

	conn, err := grpc.Dial("logger-service:50001", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Println("eror Dialing Logger service Payload")

		app.errorJson(w, r, err)
		return
	}
	defer conn.Close()

	c := logs.NewLogServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	_, err = c.WriteLog(ctx, &logs.LogRequest{
		LogEntry: &logs.Log{
			Name: requestPayload.Log.Name,
			Data: requestPayload.Log.Data,
		},
	})
	if err != nil {
		log.Println("eror Wrinting Log")

		app.errorJson(w, r, err)
		return
	}
	var payload jsonResponse
	payload.Error = false
	payload.Message = "logged"
	app.WriteJson(w, http.StatusAccepted, payload, r)
}
