# Description
CLI tool to display all the contacts per namespace on an RBAC enabled Azure Kubernetes cluster. The Azure graph API is consumed directly to query Azure contacts. 

## prerequisites
Install azure cli: https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest

Install kubectl: https://kubernetes.io/docs/tasks/tools/install-kubectl/

## create azure spn

#### set variables for creating app registration
``` shell
$ spname='<<name-spn>>'
$ tenantId=$(az account show --query tenantId -o tsv)
$ subscriptions=('<<subscription-id-01 subscription-id-02 ...>>')
```
    
#### Create the Azure AD application
``` shell
$ applicationId=$(az ad app create \
    --display-name "$spname" \
    --identifier-uris "https://$spname" \
    --query appId -o tsv)
```

#### Update the application group memebership claims
``` shell
$ az ad app update --id $applicationId --set groupMembershipClaims=All
```

#### Create a service principal for the Azure AD application
``` shell
$ az ad sp create --id $applicationId
```

#### Get the service principal secret
``` shell
$ applicationSecret=$(az ad sp credential reset \
    --name $applicationId \
    --credential-description "passwrd" \
    --query password -o tsv)
```

#### Add SPN to the subscriptions as an reader
``` shell
for s in "${subscriptions[@]}"; do {
    az role assignment create --assignee $applicationId --subscription $s --role 'Reader'
}; done
```

## set environment variables
Once the Azure App registration is created set the following environment variables:
``` shell

$ export AZURE_CLIENT_ID='$applicationId'
$ export AZURE_TENANT_ID=$tenantId
$ export AZURE_CLIENT_SECRET='$applicationSecret'
$ export K8S_KUBECONFIG='~/.kube/config' # give full path if ~ gives an error
$ export K8S_ROLEBINDING='<<name-of-rolebinding-to-export>>'
```

## install and run
``` shell
$ git clone https://github.com/bartvanbenthem/aks-contacts.git
$ cd aks-contacts/bin/
$ ./aks-contacts
```
