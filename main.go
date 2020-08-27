package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/bartvanbenthem/azuretoken"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type AzGroup struct {
	ID                           string        `json:"id"`
	DeletedDateTime              interface{}   `json:"deletedDateTime"`
	Classification               interface{}   `json:"classification"`
	CreatedDateTime              time.Time     `json:"createdDateTime"`
	CreationOptions              []interface{} `json:"creationOptions"`
	Description                  string        `json:"description"`
	DisplayName                  string        `json:"displayName"`
	GroupTypes                   []string      `json:"groupTypes"`
	Mail                         string        `json:"mail"`
	MailEnabled                  bool          `json:"mailEnabled"`
	MailNickname                 string        `json:"mailNickname"`
	OnPremisesLastSyncDateTime   interface{}   `json:"onPremisesLastSyncDateTime"`
	OnPremisesSecurityIdentifier interface{}   `json:"onPremisesSecurityIdentifier"`
	OnPremisesSyncEnabled        interface{}   `json:"onPremisesSyncEnabled"`
	PreferredDataLocation        string        `json:"preferredDataLocation"`
	ProxyAddresses               []string      `json:"proxyAddresses"`
	RenewedDateTime              time.Time     `json:"renewedDateTime"`
	ResourceBehaviorOptions      []interface{} `json:"resourceBehaviorOptions"`
	ResourceProvisioningOptions  []interface{} `json:"resourceProvisioningOptions"`
	SecurityEnabled              bool          `json:"securityEnabled"`
	Visibility                   string        `json:"visibility"`
	OnPremisesProvisioningErrors []interface{} `json:"onPremisesProvisioningErrors"`
}

type AzGroupMembers struct {
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

type K8sGroup struct {
	GroupID   string
	Namespace string
}

type Host struct {
	Group     K8sGroup
	HostName  string
	Namespace string
}

type ContactGroup struct {
	Group   K8sGroup
	Persons []string
	Owner   string
}

type K8s struct{}
type Azure struct{}

func main() {
	// Check if there are empty ENV Variables that need to be set
	CheckEmptyEnVar()

	// test hostname and contact output
	PrintHostnames()
	PrintContacts()
}

func CheckEmptyEnVar() {
	vars := []string{"AZURE_CLIENT_ID",
		"AZURE_TENANT_ID",
		"AZURE_CLIENT_SECRET",
		"K8S_KUBECONFIG",
		"K8S_ROLEBINDING"}

	for _, v := range vars {
		if os.Getenv(v) == "" {
			log.Fatalf("Fatal Error: env variable [ %v ] is empty\n", v)
		}
	}
}

func GetAllContacts(token azuretoken.GraphToken) ([]ContactGroup, error) {
	// initialize azure and kubernetes methods
	var kube K8s
	var az Azure

	var contacts []ContactGroup
	groups, err := kube.GetGroup(kube.CreateClientSet())
	if err != nil {
		return contacts, err
	}

	for _, g := range groups {
		m := az.GetGroupMembersMail(token, g.GroupID)
		c := ContactGroup{Persons: m, Group: g}
		contacts = append(contacts, c)
	}

	return contacts, err
}

func (K8s) GetCurrentContext() string {
	cmd := exec.Command("kubectl", "config", "current-context")
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error: %v\n", err)
	}

	return strings.TrimSuffix(string(stdoutStderr), "\n")
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
		if envvar := os.Getenv("K8S_KUBECONFIG"); len(envvar) > 0 {
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

func (k K8s) GetGroup(clientset *kubernetes.Clientset) ([]K8sGroup, error) {
	rbenv := os.Getenv("K8S_ROLEBINDING")
	var groups []K8sGroup

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
						group := K8sGroup{GroupID: sub.Name, Namespace: sub.Namespace}
						groups = append(groups, group)
					}
				}
			}
		}

	}
	return groups, err
}

func (k K8s) GetHostname(clientset *kubernetes.Clientset) ([]Host, error) {
	var hosts []Host

	ns, err := clientset.CoreV1().Namespaces().List(v1.ListOptions{})
	if err != nil {
		return nil, err
	}

	for _, n := range ns.Items {
		ing, err := clientset.NetworkingV1beta1().Ingresses(n.GetName()).List(v1.ListOptions{})
		if err != nil {
			return nil, err
		}

		var host Host
		for _, i := range ing.Items {
			rules := i.Spec.Rules
			for _, r := range rules {
				host.HostName = r.Host
				host.Namespace = n.GetName()
				hosts = append(hosts, host)
			}
		}
	}

	return hosts, err
}

func (a Azure) GetGroup(graphToken azuretoken.GraphToken, gid string) AzGroup {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%v", gid)
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

	var gn AzGroup
	json.Unmarshal(body, &gn)

	return gn
}

func (a Azure) GetGroupMembers(token azuretoken.GraphToken, gid string) AzGroupMembers {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%v/members", gid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error: %v\n", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
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

	var m AzGroupMembers
	json.Unmarshal(body, &m)

	return m
}

func (a Azure) GetGroupMembersMail(token azuretoken.GraphToken, gid string) []string {
	url := fmt.Sprintf("https://graph.microsoft.com/v1.0/groups/%v/members", gid)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error: %v\n", err)
	}

	req.Header.Add("Authorization", fmt.Sprintf("%s %s", token.TokenType, token.AccessToken))
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

	var m AzGroupMembers
	json.Unmarshal(body, &m)

	var members []string
	for _, v := range m.Value {
		members = append(members, v.Mail)
	}

	return members
}

func PrintHostnames() {
	var kube K8s
	// Create the hostname output
	hosts, err := kube.GetHostname(kube.CreateClientSet())
	if err != nil {
		log.Printf("Error: %v\n", err)
	}
	fmt.Printf("%-27v %-27v %v\n", "hostname", "namespace", "context")
	for _, h := range hosts {
		cluster := kube.GetCurrentContext()
		fmt.Printf("%-27v %-27v %v\n", h.HostName, h.Namespace, cluster)
	}

	fmt.Println()
}

func PrintContacts() {
	// load environment variables for Azure graph token request
	applicationid := os.Getenv("AZURE_CLIENT_ID")
	tenantid := os.Getenv("AZURE_TENANT_ID")
	secret := os.Getenv("AZURE_CLIENT_SECRET")

	// get azure graph token
	var token azuretoken.Token
	graphClient := azuretoken.GraphClient{TenantID: tenantid, ApplicationID: applicationid, ClientSecret: secret}
	gtoken := token.GetGraphToken(graphClient)

	// initiate azure and k8s methods
	var az Azure
	var kube K8s

	// Create the contact output
	contacts, err := GetAllContacts(gtoken)
	if err != nil {
		log.Printf("Error: %v\n", err)
	}

	cluster := kube.GetCurrentContext()
	fmt.Printf("%-27v %-27v %-35v %v\n", "contact", "namespace", "group", "context")
	for _, c := range contacts {
		for _, p := range c.Persons {
			gname := az.GetGroup(gtoken, c.Group.GroupID)
			fmt.Printf("%-27v %-27v %-35v %v\n", p, c.Group.Namespace, gname.DisplayName, cluster)
		}
	}
}
