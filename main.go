package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/bartvanbenthem/azuretoken"
)

type GroupMembers struct {
	OdataContext string `json:"@odata.context"`
	Value        []struct {
		OdataType         string        `json:"@odata.type"`
		ID                string        `json:"id"`
		BusinessPhones    []interface{} `json:"businessPhones"`
		DisplayName       string        `json:"displayName"`
		GivenName         string        `json:"givenName"`
		JobTitle          string        `json:"jobTitle"`
		Mail              string        `json:"mail"`
		MobilePhone       string        `json:"mobilePhone"`
		OfficeLocation    interface{}   `json:"officeLocation"`
		PreferredLanguage interface{}   `json:"preferredLanguage"`
		Surname           string        `json:"surname"`
		UserPrincipalName string        `json:"userPrincipalName"`
	} `json:"value"`
}

func main() {

	// load environment variables
	applicationid := os.Getenv("AZAPPLICATIONID")
	tenantid := os.Getenv("AZTENANT")
	secret := os.Getenv("AZSECRET")

	// get azure graph token
	var token azuretoken.Token
	graphClient := azuretoken.GraphClient{TenantID: tenantid, ApplicationID: applicationid, ClientSecret: secret}
	gtoken := token.GetGraphToken(graphClient)

	gid := "ef9ec40b-709f-49d7-93ba-48511b501a45"
	GetGroupMembers(gtoken, gid)

}

func GetGroupMembers(graphToken azuretoken.GraphToken, gid string) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%v/members", gid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("%s %s", graphToken.TokenType, graphToken.AccessToken))
	req.Header.Add("Content-Type", "application/json")
	httpClient := &http.Client{Timeout: time.Second * 10}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	var m GroupMembers
	json.Unmarshal(body, &m)

	for _, v := range m.Value {
		fmt.Println(v.Mail)
	}
}
