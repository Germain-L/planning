REGISTRY=registry.germainleignel.com/personal
VERSION=$(shell date +%Y%m%d-%H%M%S)

.PHONY: all backend frontend deploy

all: backend frontend flush deploy

backend:
	cd backend && \
	docker build -t planning-backend . && \
	docker tag planning-backend $(REGISTRY)/planning-backend:$(VERSION) && \
	docker tag planning-backend $(REGISTRY)/planning-backend:latest && \
	docker push $(REGISTRY)/planning-backend:$(VERSION) && \
	docker push $(REGISTRY)/planning-backend:latest

frontend:
	cd frontend && \
	docker build -t planning-frontend . && \
	docker tag planning-frontend $(REGISTRY)/planning-frontend:$(VERSION) && \
	docker tag planning-frontend $(REGISTRY)/planning-frontend:latest && \
	docker push $(REGISTRY)/planning-frontend:$(VERSION) && \
	docker push $(REGISTRY)/planning-frontend:latest

deploy:
	kubectl apply -f k8s/
	kubectl rollout restart deployment/planning-frontend -n planning
	kubectl rollout restart deployment/planning-backend -n planning