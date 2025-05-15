package main

import (
	"log"
	"net/http"
	"os"
)

func (app *Config) SendMail(w http.ResponseWriter, r *http.Request) {
	type mailMessage struct {
		From    string `json:from`
		To      string `json:to`
		Subject string `json:subject`
		Message string `json:messge`
	}

	var requestPayload mailMessage

	err := app.readJson(w, r, &requestPayload)
	if err != nil {
		log.Println("Reading JSON", err)
		app.errorJson(w, r, err)
	}

	msg := Message{
		From:    requestPayload.From,
		To:      requestPayload.To,
		Subject: requestPayload.Subject,
		Data:    requestPayload.Message,
	}

	err = app.Mailer.sendSMTPMessage(msg)
	if err != nil {
		log.Println("SENDING SMPT", err)
		files, _ := os.ReadDir(".")
		for _, f := range files {
			log.Println(f.Name())
		}
		app.errorJson(w, r, err)
	}

	payload := jsonResponse{
		Error:   false,
		Message: "Sent to " + requestPayload.To,
	}
	app.WriteJson(w, http.StatusAccepted, payload, r)

}
