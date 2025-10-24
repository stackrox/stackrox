# Makefile for ML Risk Service

# Variables
DOCKER_IMAGE ?= quay.io/stehessel/ml-risk-service
DOCKER_TAG ?= latest
NAMESPACE ?= stackrox
K8S_CONTEXT ?= $(shell kubectl config current-context)

# Colors for output
RED := \033[0;31m
GREEN := \033[0;32m
YELLOW := \033[0;33m
BLUE := \033[0;34m
NC := \033[0m # No Color

.PHONY: help
help: ## Show this help message
	@echo "ML Risk Service - Available Commands:"
	@echo
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(BLUE)%-20s$(NC) %s\n", $$1, $$2}'
	@echo

# Development commands
.PHONY: setup
setup: ## Set up development environment
	@echo "$(GREEN)Setting up development environment...$(NC)"
	python -m pip install --upgrade pip
	pip install -r requirements.txt
	@echo "$(GREEN)Development environment ready!$(NC)"

.PHONY: test
test: ## Run tests
	@echo "$(GREEN)Running tests...$(NC)"
	python -m pytest tests/ -v --cov=src --cov-report=html
	@echo "$(GREEN)Tests completed!$(NC)"

.PHONY: lint
lint: ## Run linting
	@echo "$(GREEN)Running linting...$(NC)"
	python -m flake8 src/ training/ --max-line-length=120
	python -m pylint src/ training/ || true
	@echo "$(GREEN)Linting completed!$(NC)"

.PHONY: format
format: ## Format code
	@echo "$(GREEN)Formatting code...$(NC)"
	python -m black src/ training/
	python -m isort src/ training/
	@echo "$(GREEN)Code formatted!$(NC)"

.PHONY: generate-sample-data
generate-sample-data: ## Generate sample training data
	@echo "$(GREEN)Generating sample training data...$(NC)"
	python -c "from training.data_loader import JSONTrainingDataGenerator; g = JSONTrainingDataGenerator(); g.generate_sample_training_data('sample_training_data.json', 1000)"
	@echo "$(GREEN)Sample data generated: sample_training_data.json$(NC)"

.PHONY: train-model
train-model: ## Train ML model with sample data
	@echo "$(GREEN)Training ML model...$(NC)"
	python -c "from training.train_pipeline import TrainingPipeline; tp = TrainingPipeline(); tp.quick_test_pipeline()"
	@echo "$(GREEN)Model training completed!$(NC)"

# Docker commands
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(GREEN)Building Docker image...$(NC)"
	docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

.PHONY: docker-run
docker-run: ## Run Docker container locally
	@echo "$(GREEN)Starting Docker container...$(NC)"
	docker run -d \
		--name ml-risk-service \
		-p 8080:8080 \
		-p 8081:8081 \
		-v $(PWD)/models:/app/models \
		$(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)Container started! gRPC: http://localhost:8080, Health: http://localhost:8081$(NC)"

.PHONY: docker-stop
docker-stop: ## Stop Docker container
	@echo "$(YELLOW)Stopping Docker container...$(NC)"
	docker stop ml-risk-service || true
	docker rm ml-risk-service || true
	@echo "$(GREEN)Container stopped!$(NC)"

.PHONY: docker-logs
docker-logs: ## Show Docker container logs
	docker logs -f ml-risk-service

.PHONY: docker-push
docker-push: ## Push Docker image to registry
	@echo "$(GREEN)Pushing Docker image...$(NC)"
	docker push $(DOCKER_IMAGE):$(DOCKER_TAG)
	@echo "$(GREEN)Image pushed!$(NC)"

# Docker Compose commands
.PHONY: compose-up
compose-up: ## Start services with Docker Compose
	@echo "$(GREEN)Starting services with Docker Compose...$(NC)"
	docker-compose up -d
	@echo "$(GREEN)Services started! ML Service: http://localhost:8080, Monitoring: http://localhost:3000$(NC)"

.PHONY: compose-down
compose-down: ## Stop Docker Compose services
	@echo "$(YELLOW)Stopping Docker Compose services...$(NC)"
	docker-compose down
	@echo "$(GREEN)Services stopped!$(NC)"

.PHONY: compose-logs
compose-logs: ## Show Docker Compose logs
	docker-compose logs -f ml-risk-service

# Kubernetes commands
.PHONY: k8s-deploy
k8s-deploy: ## Deploy to Kubernetes
	@echo "$(GREEN)Deploying to Kubernetes (context: $(K8S_CONTEXT))...$(NC)"
	kubectl apply -k k8s/
	@echo "$(GREEN)Deployed to Kubernetes!$(NC)"

.PHONY: k8s-delete
k8s-delete: ## Delete from Kubernetes
	@echo "$(YELLOW)Deleting from Kubernetes...$(NC)"
	kubectl delete -k k8s/ || true
	@echo "$(GREEN)Deleted from Kubernetes!$(NC)"

.PHONY: k8s-status
k8s-status: ## Show Kubernetes deployment status
	@echo "$(BLUE)Kubernetes Status:$(NC)"
	kubectl get pods,svc,deploy -n $(NAMESPACE) -l app.kubernetes.io/name=ml-risk-service

.PHONY: k8s-logs
k8s-logs: ## Show Kubernetes pod logs
	kubectl logs -f -n $(NAMESPACE) -l app.kubernetes.io/name=ml-risk-service

.PHONY: k8s-port-forward
k8s-port-forward: ## Port forward to Kubernetes service
	@echo "$(GREEN)Port forwarding to Kubernetes service...$(NC)"
	kubectl port-forward -n $(NAMESPACE) svc/ml-risk-service 8080:8080 8081:8081

.PHONY: k8s-shell
k8s-shell: ## Get shell in Kubernetes pod
	kubectl exec -it -n $(NAMESPACE) $$(kubectl get pods -n $(NAMESPACE) -l app.kubernetes.io/name=ml-risk-service -o jsonpath='{.items[0].metadata.name}') -- /bin/bash

# Health check commands
.PHONY: health-check
health-check: ## Check service health
	@echo "$(BLUE)Checking service health...$(NC)"
	curl -f http://localhost:8081/health || echo "$(RED)Health check failed$(NC)"
	curl -f http://localhost:8081/ready || echo "$(RED)Readiness check failed$(NC)"

.PHONY: test-prediction
test-prediction: ## Test prediction endpoint (requires grpcurl)
	@echo "$(BLUE)Testing prediction endpoint...$(NC)"
	@which grpcurl > /dev/null || (echo "$(RED)grpcurl not found. Install with: go install github.com/fullstorydev/grpcurl/cmd/grpcurl@latest$(NC)" && exit 1)
	grpcurl -plaintext localhost:8080 ml_risk.MLRiskService/GetModelHealth

# Monitoring commands
.PHONY: monitoring-up
monitoring-up: ## Start monitoring stack
	@echo "$(GREEN)Starting monitoring stack...$(NC)"
	mkdir -p monitoring/prometheus monitoring/grafana/dashboards monitoring/grafana/datasources
	docker-compose up -d prometheus grafana
	@echo "$(GREEN)Monitoring available at: http://localhost:3000 (admin/admin)$(NC)"

.PHONY: monitoring-down
monitoring-down: ## Stop monitoring stack
	docker-compose stop prometheus grafana

# Cleanup commands
.PHONY: clean
clean: ## Clean up temporary files
	@echo "$(YELLOW)Cleaning up...$(NC)"
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} + 2>/dev/null || true
	rm -rf .coverage htmlcov/ .pytest_cache/
	@echo "$(GREEN)Cleanup completed!$(NC)"

.PHONY: clean-docker
clean-docker: ## Clean up Docker resources
	@echo "$(YELLOW)Cleaning up Docker resources...$(NC)"
	docker-compose down --volumes --remove-orphans || true
	docker system prune -f
	@echo "$(GREEN)Docker cleanup completed!$(NC)"

# Development workflow
.PHONY: dev-setup
dev-setup: setup generate-sample-data ## Complete development setup
	@echo "$(GREEN)Development setup completed!$(NC)"
	@echo "Next steps:"
	@echo "  1. Run 'make train-model' to train a sample model"
	@echo "  2. Run 'make docker-build && make docker-run' to test locally"
	@echo "  3. Run 'make k8s-deploy' to deploy to Kubernetes"

.PHONY: ci-pipeline
ci-pipeline: lint test docker-build ## Run CI pipeline
	@echo "$(GREEN)CI pipeline completed successfully!$(NC)"

# Production deployment
.PHONY: deploy-prod
deploy-prod: docker-build docker-push k8s-deploy ## Deploy to production
	@echo "$(GREEN)Production deployment completed!$(NC)"

# Default target
.DEFAULT_GOAL := help
