kubectl apply -f charles-deployment-crd.yaml && kubectl create namespace dev && kubectl apply -f charles-deployment.yaml -n dev && go run main.go