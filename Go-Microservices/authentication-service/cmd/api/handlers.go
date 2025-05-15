package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

func (app *Config) Authenticate(w http.ResponseWriter, r *http.Request) {
	var requestPayload struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	err := app.readJson(w, r, &requestPayload)

	if err != nil {
		log.Println("readJson ERROR")

		app.errorJson(w, r, err, http.StatusBadRequest)
		return
	}

	//validate the useer against the dababse
	user, err := app.Models.User.GetByEmail(requestPayload.Email)
	if err != nil {
		log.Println("Get BY EMAIL ERROR")
		app.errorJson(w, r, errors.New("InvalidCredentials"), http.StatusBadRequest)
		return
	}

	valid, err := user.PasswordMatches(requestPayload.Password)
	if err != nil || !valid {
		log.Println("PasswordMatches ERROR")

		app.errorJson(w, r, errors.New("InvalidCredentials"), http.StatusBadRequest)
		return
	}

	//log Authentication
	err = app.logRequest("authentication", fmt.Sprintf("%s logged in", user.Email))
	if err != nil {
		log.Println("Logging  ERROR")

		app.errorJson(w, r, errors.New("Unale to log"), http.StatusBadRequest)
		return
	}
	payload := jsonResponse{
		Error:   false,
		Message: fmt.Sprintf("Logged in user %s", user.Email),
		Data:    user,
	}
	app.WriteJson(w, http.StatusAccepted, payload, r)
}

func (app *Config) logRequest(name, data string) error {
	var entry struct {
		Name string `json:"name"`
		Data string `json:"data"`
	}
	entry.Name = name
	entry.Data = data

	jsonData, _ := json.MarshalIndent(entry, "", "\t")
	logServiceURL := "http://logger-service/log"
	request, err := http.NewRequest("POST", logServiceURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	client := &http.Client{}
	_, err = client.Do(request)
	if err != nil {
		return err
	}
	return nil
}
