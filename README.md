# Description
display all the contacts per namespace or listner on an RBAC enabled Azure Kubernetes cluster.


# create azure spn

### set variables for creating app registration
`
* spname='<name-spn>'
* tenantId=$(az account show --query tenantId -o tsv)
* subscriptions=('<subscription-id>')
`
    
### Create the Azure AD application
`
applicationId=$(az ad app create \
    --display-name "$spname" \
    --identifier-uris "https://$spname" \
    --query appId -o tsv)
`

### Update the application group memebership claims
`az ad app update --id $applicationId --set groupMembershipClaims=All`

### Create a service principal for the Azure AD application
`az ad sp create --id $applicationId`

### Get the service principal secret
`
applicationSecret=$(az ad sp credential reset \
    --name $applicationId \
    --credential-description "golangpass" \
    --query password -o tsv)
`
### sleep
`echo "waiting for app to be ready for the role assignments"`
`sleep 10`

### Add SPN to the subscriptions as an reader
`
for s in "${subscriptions[@]}"; do {
    az role assignment create --assignee $applicationId --subscription $s --role 'Reader'
}; done
`

# set env vars
Once the Azure App registration is created set the following environment variables:
```
export AZAPPLICATIONID='<spn-id>'
export AZTENANT='<azure-tenant-id>'
export AZSECRET='<spn-secret>'
export KUBECONFIG='~/.kube/config'
export ROLEBINDING='<name-of-rolebinding>'
```
