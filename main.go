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

type Group struct {
	GroupID   string
	GroupName string
	Namespace string
}

type Contacts struct {
	Persons []string
	Group   Group
}

type ListnerContactInfo struct {
	HostName       string
	ListnerName    string
	ContactPersons []Contacts
	Cluster        string
}

type K8s struct{}
type Azure struct{}

func main() {

	// initialize azure and kubernetes methods
	var kube K8s
	var az Azure

	// load environment variables for Azure environment
	applicationid := os.Getenv("AZAPPLICATIONID")
	tenantid := os.Getenv("AZTENANT")
	secret := os.Getenv("AZSECRET")

	// get azure graph token
	var token azuretoken.Token
	graphClient := azuretoken.GraphClient{TenantID: tenantid, ApplicationID: applicationid, ClientSecret: secret}
	gtoken := token.GetGraphToken(graphClient)

	// run the GetListnerInfo function to create ListnerInfo output object
	groups, err := kube.GetGroup(kube.CreateClientSet())
	if err != nil {
		log.Printf("Error: %v\n", err)
	}

	for _, g := range groups {
		az.GetGroupMembers(gtoken, g.GroupID)
	}

}

func (k K8s) GetGroup(clientset *kubernetes.Clientset) ([]Group, error) {
	rbenv := os.Getenv("ROLEBINDING")
	var groups []Group

	ns, err := clientset.CoreV1().Namespaces().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, n := range ns.Items {
		rb, err := clientset.RbacV1().RoleBindings(n.GetName()).List(v1.ListOptions{})
		if err != nil {
			return nil, err
		}

		for _, r := range rb.Items {
			if r.GetName() == rbenv {
				for _, sub := range r.Subjects {
					if sub.APIGroup == "rbac.authorization.k8s.io" {
						group := Group{GroupID: sub.Name, Namespace: sub.Namespace}
						groups = append(groups, group)
					}
				}
			}
		}

	}
	return groups, err
}

func (K8s) CreateClientSet() *kubernetes.Clientset {
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
		log.Printf("Error: %v\n", err)
	}

	return clientset
}

func (a Azure) GetGroup() {}

func (a Azure) GetGroupMembers(graphToken azuretoken.GraphToken, gid string) {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%v/members", gid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error: %v\n", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("%s %s", graphToken.TokenType, graphToken.AccessToken))
	req.Header.Add("Content-Type", "application/json")
	httpClient := &http.Client{Timeout: time.Second * 10}
	resp, err := httpClient.Do(req)
	if err != nil {
		log.Printf("Error: %v\n", err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Error: %v\n", err)
	}

	var m GroupMembers
	json.Unmarshal(body, &m)

	for _, v := range m.Value {
		fmt.Println(v.Mail)
	}
}
