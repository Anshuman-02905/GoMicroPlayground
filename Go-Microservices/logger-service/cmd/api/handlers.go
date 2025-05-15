package main

import (
	"log"
	"log-service/data"
	"net/http"
)

type JSONPayload struct {
	Name string `json:"name"`
	Data string `json:"data"`
}

func (app *Config) WriteLog(w http.ResponseWriter, r *http.Request) {
	//read json inot var
	var requestPayLoad JSONPayload
	_ = app.readJson(w, r, &requestPayLoad)

	//insert data
	event := data.LogEntry{
		Name: requestPayLoad.Name,
		Data: requestPayLoad.Data,
	}

	err := app.Models.LogEntry.Insert(event)

	if err != nil {
		app.errorJson(w, r, err)
		log.Println("Error inserting log into Mongo", "*****", event, "******")
		return
	}
	resp := jsonResponse{
		Error:   false,
		Message: "logged",
	}

	app.WriteJson(w, http.StatusAccepted, resp, r)
}
