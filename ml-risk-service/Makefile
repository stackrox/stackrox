# Makefile for ML Risk Service

# Variables
DOCKER_IMAGE ?= quay.io/stehessel/ml-risk-service
DOCKER_TAG ?= latest
NAMESPACE ?= stackrox
K8S_CONTEXT ?= $(shell kubectl config current-context)

# Colors for output (compatible with kitty terminal)
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
BOLD := \033[1m
NC := \033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo -e "$(BOLD)ML Risk Service - Available Commands:$(NC)"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo

.PHONY: test-colors
test-colors: ## Test color output in terminal
	@echo -e "$(RED)Red text$(NC)"
	@echo -e "$(GREEN)Green text$(NC)"
	@echo -e "$(YELLOW)Yellow text$(NC)"
	@echo -e "$(BLUE)Blue text$(NC)"
	@echo -e "$(BOLD)Bold text$(NC)"

# Development commands
.PHONY: setup
setup: ## Set up development environment
	@echo -e "$(GREEN)Setting up development environment...$(NC)"
	python -m pip install --upgrade pip
	pip install -r requirements.txt
	@echo -e "$(GREEN)Development environment ready!$(NC)"

.PHONY: test
test: ## Run tests
	@echo -e "$(GREEN)Running tests...$(NC)"
	python -m pytest tests/ -v --cov=src --cov-report=html
	@echo -e "$(GREEN)Tests completed!$(NC)"

.PHONY: lint
lint: ## Run linting
	@echo -e "$(GREEN)Running linting...$(NC)"
	python -m flake8 src/ training/ --max-line-length=120
	python -m pylint src/ training/ || true
	@echo -e "$(GREEN)Linting completed!$(NC)"

.PHONY: format
format: ## Format code
	@echo -e "$(GREEN)Formatting code...$(NC)"
	python -m black src/ training/
	python -m isort src/ training/
	@echo -e "$(GREEN)Code formatted!$(NC)"

.PHONY: generate-sample-data
generate-sample-data: ## Generate sample training data
	@echo -e "$(GREEN)Generating sample training data...$(NC)"
	python -c "from training.data_loader import JSONTrainingDataGenerator; g = JSONTrainingDataGenerator(); g.generate_sample_training_data('sample_training_data.json', 1000)"
	@echo -e "$(GREEN)Sample data generated: sample_training_data.json$(NC)"

.PHONY: train-model
train-model: ## Train ML model with sample data
	@echo -e "$(GREEN)Training ML model...$(NC)"
	python -c "from training.train_pipeline import TrainingPipeline; tp = TrainingPipeline(); tp.quick_test_pipeline()"
	@echo -e "$(GREEN)Model training completed!$(NC)"

# Configuration commands
.PHONY: generate-config
generate-config: ## Generate config with expanded environment variables
	@echo -e "$(GREEN)Generating expanded configuration...$(NC)"
	@# Set defaults for missing variables to prevent envsubst errors
	@export CENTRAL_API_TOKEN="$${CENTRAL_API_TOKEN:-}" && \
	export CENTRAL_CLIENT_CERT_PATH="$${CENTRAL_CLIENT_CERT_PATH:-}" && \
	export CENTRAL_CLIENT_KEY_PATH="$${CENTRAL_CLIENT_KEY_PATH:-}" && \
	export CENTRAL_CA_CERT_PATH="$${CENTRAL_CA_CERT_PATH:-}" && \
	envsubst < src/config/feature_config.yaml > src/config/feature_config.expanded.yaml
	@echo -e "$(GREEN)Configuration expanded: src/config/feature_config.expanded.yaml$(NC)"

# Docker commands
.PHONY: docker-build
docker-build: generate-config ## Build Docker image
	@echo -e "$(GREEN)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo -e "$(GREEN)Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

.PHONY: docker-run
docker-run: ## Run Docker container locally
	@echo -e "$(GREEN)Starting Docker container...$(NC)"
	@# Stop and remove existing container if it exists
	@docker stop ml-risk-service 2>/dev/null || true
	@docker rm ml-risk-service 2>/dev/null || true
	@# Ensure directories exist
	@mkdir -p $(PWD)/deploy/models $(PWD)/deploy/reports
	@# Fix permissions for container user
	@chmod 777 $(PWD)/deploy/models $(PWD)/deploy/reports
	@# Run the container
	docker run -d \
		--name ml-risk-service \
		-p 8080:8080 \
		-p 8081:8081 \
		-p 8090:8090 \
		-v $(PWD)/deploy/models:/app/models \
		-v $(PWD)/deploy/reports:/app/reports \
		-e LOG_LEVEL=INFO \
		-e GRPC_PORT=8080 \
		-e HEALTH_PORT=8081 \
		-e REST_PORT=8090 \
		-e ENABLE_REST=true \
		--restart unless-stopped \
		$(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo -e "$(GREEN)Container started!$(NC)"
	@echo -e "$(BLUE)  - gRPC API: localhost:8080$(NC)"
	@echo -e "$(BLUE)  - Health: http://localhost:8081$(NC)"
	@echo -e "$(BLUE)  - REST API: http://localhost:8090$(NC)"
	@echo -e "$(BLUE)  - API Docs: http://localhost:8090/docs$(NC)"
	@echo -e "$(BLUE)Check logs with: make docker-logs$(NC)"

.PHONY: docker-stop
docker-stop: ## Stop Docker container
	@echo -e "$(YELLOW)Stopping Docker container...$(NC)"
	docker stop ml-risk-service || true
	docker rm ml-risk-service || true
	@echo -e "$(GREEN)Container stopped!$(NC)"

.PHONY: docker-logs
docker-logs: ## Show Docker container logs
	docker logs -f ml-risk-service

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@echo -e "$(GREEN)Pushing Docker image...$(NC)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo -e "$(GREEN)Image pushed!$(NC)"

# Docker Compose commands
.PHONY: compose-up
compose-up: ## Start services with Docker Compose
	@echo -e "$(GREEN)Starting services with Docker Compose...$(NC)"
	docker-compose up -d
	@echo -e "$(GREEN)Services started! ML Service: http://localhost:8080, Monitoring: http://localhost:3000$(NC)"

.PHONY: compose-down
compose-down: ## Stop Docker Compose services
	@echo -e "$(YELLOW)Stopping Docker Compose services...$(NC)"
	docker-compose down
	@echo -e "$(GREEN)Services stopped!$(NC)"

.PHONY: compose-logs
compose-logs: ## Show Docker Compose logs
	docker-compose logs -f ml-risk-service

# Kubernetes commands
.PHONY: k8s-deploy
k8s-deploy: ## Deploy to Kubernetes
	@echo -e "$(GREEN)Deploying to Kubernetes (context: $(K8S_CONTEXT))...$(NC)"
	kubectl apply -k k8s/
	@echo -e "$(GREEN)Deployed to Kubernetes!$(NC)"

.PHONY: k8s-delete
k8s-delete: ## Delete from Kubernetes
	@echo -e "$(YELLOW)Deleting from Kubernetes...$(NC)"
	kubectl delete -k k8s/ || true
	@echo -e "$(GREEN)Deleted from Kubernetes!$(NC)"

.PHONY: k8s-status
k8s-status: ## Show Kubernetes deployment status
	@echo -e "$(BLUE)Kubernetes Status:$(NC)"
	kubectl get pods,svc,deploy -n $(NAMESPACE) -l app.kubernetes.io/name=ml-risk-service

.PHONY: k8s-logs
k8s-logs: ## Show Kubernetes pod logs
	kubectl logs -f -n $(NAMESPACE) -l app.kubernetes.io/name=ml-risk-service

.PHONY: k8s-port-forward
k8s-port-forward: ## Port forward to Kubernetes service
	@echo -e "$(GREEN)Port forwarding to Kubernetes service...$(NC)"
	@echo -e "$(BLUE)Access endpoints at:$(NC)"
	@echo -e "$(BLUE)  - gRPC: localhost:8080$(NC)"
	@echo -e "$(BLUE)  - Health: http://localhost:8081$(NC)"
	@echo -e "$(BLUE)  - REST API: http://localhost:8090$(NC)"
	@echo -e "$(BLUE)  - API Docs: http://localhost:8090/docs$(NC)"
	kubectl port-forward -n $(NAMESPACE) svc/ml-risk-service 8080:8080 8081:8081 8090:8090

.PHONY: k8s-shell
k8s-shell: ## Get shell in Kubernetes pod
	kubectl exec -it -n $(NAMESPACE) $$(kubectl get pods -n $(NAMESPACE) -l app.kubernetes.io/name=ml-risk-service -o jsonpath='{.items[0].metadata.name}') -- /bin/bash

# Health check commands
.PHONY: health-check
health-check: ## Check service health
	@echo -e "$(BLUE)Checking service health...$(NC)"
	curl -f http://localhost:8081/health || echo -e "$(RED)Health check failed$(NC)"
	curl -f http://localhost:8081/ready || echo -e "$(RED)Readiness check failed$(NC)"

.PHONY: test-prediction
test-prediction: ## Test prediction endpoint (requires grpcurl)
	@echo -e "$(BLUE)Testing prediction endpoint...$(NC)"
	@which grpcurl > /dev/null || (echo -e "$(RED)grpcurl not found. Install with: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest$(NC)" && exit 1)
	grpcurl -plaintext localhost:8080 ml_risk.MLRiskService/GetModelHealth

# Monitoring commands
.PHONY: monitoring-up
monitoring-up: ## Start monitoring stack
	@echo -e "$(GREEN)Starting monitoring stack...$(NC)"
	mkdir -p monitoring/prometheus monitoring/grafana/dashboards monitoring/grafana/datasources
	docker-compose up -d prometheus grafana
	@echo -e "$(GREEN)Monitoring available at: http://localhost:3000 (admin/admin)$(NC)"

.PHONY: monitoring-down
monitoring-down: ## Stop monitoring stack
	docker-compose stop prometheus grafana

# Cleanup commands
.PHONY: clean
clean: ## Clean up temporary files
	@echo -e "$(YELLOW)Cleaning up...$(NC)"
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} + 2>/dev/null || true
	rm -rf .coverage htmlcov/ .pytest_cache/
	rm -f src/config/feature_config.expanded.yaml
	@echo -e "$(GREEN)Cleanup completed!$(NC)"

.PHONY: clean-docker
clean-docker: ## Clean up Docker resources
	@echo -e "$(YELLOW)Cleaning up Docker resources...$(NC)"
	docker-compose down --volumes --remove-orphans || true
	docker system prune -f
	@echo -e "$(GREEN)Docker cleanup completed!$(NC)"

# Development workflow
.PHONY: dev-setup
dev-setup: setup generate-sample-data ## Complete development setup
	@echo -e "$(GREEN)Development setup completed!$(NC)"
	@echo "Next steps:"
	@echo "  1. Run 'make train-model' to train a sample model"
	@echo "  2. Run 'make docker-build && make docker-run' to test locally"
	@echo "  3. Run 'make k8s-deploy' to deploy to Kubernetes"

.PHONY: ci-pipeline
ci-pipeline: lint test docker-build ## Run CI pipeline
	@echo -e "$(GREEN)CI pipeline completed successfully!$(NC)"

# Production deployment
.PHONY: deploy-prod
deploy-prod: docker-build docker-push k8s-deploy ## Deploy to production
	@echo -e "$(GREEN)Production deployment completed!$(NC)"

# Default target
.DEFAULT_GOAL := help
