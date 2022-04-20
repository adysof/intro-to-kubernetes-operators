all: environment load-images

.PHONY: kind ingress-nginx

environment: kind ingress-nginx

kind:

	kind create cluster --config kind.yaml

destroy:

	kind delete cluster --name operators

ingress-nginx: ingress-nginx-install

ingress-nginx-install:

	kubectl apply -f nginx-deploy.yaml	

load-images:

	kind load docker-image adysof/grav --name operators
	kind load docker-image k8s.gcr.io/ingress-nginx/controller:v1.1.0@sha256:f766669fdcf3dc26347ed273a55e754b427eb4411ee075a53f30718b4499076a --name operators
