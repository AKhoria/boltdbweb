#!/bin/bash

# Set the namespace and other variables
NAMESPACE=kasten-io
DEPLOYMENT_NAME=catalog-svc
PVC_NAME=catalog-pv-claim
POD_NAME=catalog-explorer-pod
CONTAINER_IMAGE=gcr.io/rich-access-174020/catalog-explorer:latest
LOCAL_PORT=8080
POD_PORT=8080

# Function to connect to the catalog
connect_to_catalog() {
    echo "Scaling down deployment $DEPLOYMENT_NAME to zero replicas..."
    kubectl scale deployment $DEPLOYMENT_NAME --replicas=0 -n $NAMESPACE

    echo "Waiting for deployment $DEPLOYMENT_NAME to scale down..."
    kubectl rollout status deployment $DEPLOYMENT_NAME -n $NAMESPACE

    echo "Starting a new pod $POD_NAME with PVC $PVC_NAME..."
    kubectl run $POD_NAME --image=$CONTAINER_IMAGE --restart=Never --namespace=$NAMESPACE --overrides='
    {
      "apiVersion": "v1",
      "kind": "Pod",
      "metadata": {
        "name": "'$POD_NAME'",
        "namespace": "'$NAMESPACE'"
      },
      "spec": {
        "containers": [
          {
            "name": "'$POD_NAME'",
            "image": "'$CONTAINER_IMAGE'",
            "volumeMounts": [
              {
                "mountPath": "/data",
                "name": "data-volume"
              }
            ],
            "command": ["./boltdbweb", "-d", "/data/kasten-io/catalog/model-store.db"],
            "ports": [
                    {
                        "containerPort": 8080,
                        "protocol": "TCP"
                    }
                ]
          }
        ],
        "volumes": [
          {
            "name": "data-volume",
            "persistentVolumeClaim": {
              "claimName": "'$PVC_NAME'"
            }
          }
        ]
      }
    }'

    echo "Pod $POD_NAME created successfully."

    wait_for_pod_running

    echo "Port forwarding from local port $LOCAL_PORT to pod port $POD_PORT..."
    kubectl port-forward pod/$POD_NAME $LOCAL_PORT:$POD_PORT -n $NAMESPACE &
    PORT_FORWARD_PID=$!
    echo "Port forwarding set up with PID $PORT_FORWARD_PID."
}

# Function to disconnect from the catalog
disconnect() {
    echo "Terminating port forwarding with PID $PORT_FORWARD_PID..."
    kill $PORT_FORWARD_PID

    echo "Deleting pod $POD_NAME..."
    kubectl delete pod $POD_NAME -n $NAMESPACE

    echo "Scaling up deployment $DEPLOYMENT_NAME back to 1 replica..."
    kubectl scale deployment $DEPLOYMENT_NAME --replicas=1 -n $NAMESPACE

    echo "Waiting for deployment $DEPLOYMENT_NAME to scale up..."
    kubectl rollout status deployment $DEPLOYMENT_NAME -n $NAMESPACE

    echo "Catalog service is now reconnected and running."
}

wait_for_pod_running() {
    echo "Waiting for pod $POD_NAME to be in running state..."
    until [ "$(kubectl get pod $POD_NAME -n $NAMESPACE -o jsonpath='{.status.phase}')" == "Running" ]; do
        sleep 2
    done
    echo "Pod $POD_NAME is now running."
}

# Check command line arguments
if [ "$1" == "connect" ]; then
    connect_to_catalog
elif [ "$1" == "disconnect" ]; then
    disconnect
else
    echo "Usage: $0 {connect|disconnect}"
    exit 1
fi
