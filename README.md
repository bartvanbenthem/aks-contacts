# Description
display all the contacts per namespace or listner on an RBAC enabled Azure Kubernetes cluster.


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

## set env vars
Once the Azure App registration is created set the following environment variables:
``` shell
$ export AZAPPLICATIONID='$applicationId'
$ export AZTENANT=$tenantId
$ export AZSECRET='$applicationSecret'
$ export KUBECONFIG='~/.kube/config' # give full path if ~ gives an error
$ export ROLEBINDING='<<name-of-rolebinding-to-export>>'
```

## run binary
``` shell
$ git clone https://github.com/bartvanbenthem/aks2contact.git
$ ./bin/aks2contact
```
