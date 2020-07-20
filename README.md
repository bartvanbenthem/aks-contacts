# aks2contact
display all the contacts per namespace or listner on an RBAC enabled Kubernetes cluster


# create azure spn


# set env vars

Once the Azure App registration is created set the following environment variables:

export AZAPPLICATIONID='<spn-id>'
export AZTENANT='<azure-tenant-id>'
export AZSECRET='<spn-secret>'
export KUBECONFIG='~/.kube/config'
export ROLEBINDING='<name-of-rolebinding>'
