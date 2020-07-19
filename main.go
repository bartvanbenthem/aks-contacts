package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/bartvanbenthem/azuretoken"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
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

type ContactInfo struct {
	GroupID        string
	GroupName      string
	Namespace      string
	Cluster        string
	ContactPersons []string
	SolverGroup    string
	Application    string
}

func main() {

	// load environment variables for Azure environment
	applicationid := os.Getenv("AZAPPLICATIONID")
	tenantid := os.Getenv("AZTENANT")
	secret := os.Getenv("AZSECRET")

	// get azure graph token
	var token azuretoken.Token
	graphClient := azuretoken.GraphClient{TenantID: tenantid, ApplicationID: applicationid, ClientSecret: secret}
	gtoken := token.GetGraphToken(graphClient)

	gid := "ef9ec40b-709f-49d7-93ba-48511b501a45" // azure adgroup id
	GetGroupMembers(gtoken, gid)

	// run the GetListnerInfo function to create ListnerInfo output object
	listners, err := GetListnerInfo(CreateClientSet())
	if err != nil {
		log.Printf("Error: %v", err)
	}

	for _, l := range listners {
		fmt.Printf("%v,%v,%v,%v,%v,%v,%v\n", l.HostName,
			l.Name, l.Namespace, l.Application,
			l.OpsTeam, l.Department, l.Cluster)
	}

}

func CreateClientSet() *kubernetes.Clientset {
	// When running the binary inside of a pod in a cluster,
	// the kubelet will automatically mount a service account into the container at:
	// /var/run/secrets/kubernetes.io/serviceaccount.
	// It replaces the kubeconfig file and is turned into a rest.Config via the rest.InClusterConfig() method
	config, err := rest.InClusterConfig()
	if err != nil {
		// fallback to kubeconfig
		kubeconfig := filepath.Join("~", ".kube", "config")
		if envvar := os.Getenv("KUBECONFIG"); len(envvar) > 0 {
			kubeconfig = envvar
		}
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			fmt.Printf("The kubeconfig cannot be loaded: %v\n", err)
			os.Exit(1)
		}
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Printf("Error: %v", err)
	}

	return clientset
}

func GetGroup() {}

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

//////////////////////////////////////////////////////////////////////////

type ListnerInfo struct {
	Name        string
	HostName    string
	ServiceName string
	Department  string
	OpsTeam     string
	Application string
	Namespace   string
	Cluster     string
}

func GetListnerInfo(clientset *kubernetes.Clientset) ([]ListnerInfo, error) {
	// initiate ListnerInfo output objects
	var listner ListnerInfo
	var listners []ListnerInfo

	// access the API to list ingress resources
	ing, err := clientset.NetworkingV1beta1().Ingresses("").List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, i := range ing.Items {
		listner.Cluster = "cluster-name"

		// access the API to get Namespace label resources
		ns, err := clientset.CoreV1().Namespaces().Get(i.Namespace, v1.GetOptions{})
		if err != nil {
			return nil, err
		}
		labels := ns.ObjectMeta.Labels
		listner.Department = labels[os.Getenv("DEPARTMENTTAG")]
		listner.Application = labels[os.Getenv("APPLICATIONTAG")]
		listner.OpsTeam = labels[os.Getenv("OPSTAG")]

		// access the API to get ingress resources
		ir, err := clientset.NetworkingV1beta1().Ingresses(i.Namespace).Get(i.Name, v1.GetOptions{})
		if err != nil {
			return nil, err
		}

		rules := ir.Spec.Rules
		for _, r := range rules {
			listner.Name = i.Name
			listner.HostName = r.Host
			listner.Namespace = i.Namespace
			listners = append(listners, listner)
		}
	}

	return listners, err
}
