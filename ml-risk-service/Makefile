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
setup: ## Set up development environment with UV
	@echo -e "$(GREEN)Setting up development environment with UV...$(NC)"
	@which uv > /dev/null || (echo -e "$(RED)UV not found. Install with: curl -LsSf https://astral.sh/uv/install.sh | sh$(NC)" && exit 1)
	uv sync
	@echo -e "$(GREEN)Development environment ready!$(NC)"

.PHONY: setup-dev
setup-dev: ## Set up development environment with dev dependencies
	@echo -e "$(GREEN)Setting up development environment with dev dependencies...$(NC)"
	@which uv > /dev/null || (echo -e "$(RED)UV not found. Install with: curl -LsSf https://astral.sh/uv/install.sh | sh$(NC)" && exit 1)
	uv sync --extra dev
	@echo -e "$(GREEN)Development environment with dev tools ready!$(NC)"

.PHONY: setup-ml
setup-ml: ## Set up with extended ML dependencies
	@echo -e "$(GREEN)Setting up with extended ML dependencies...$(NC)"
	uv sync --extra ml --extra dev
	@echo -e "$(GREEN)ML environment ready!$(NC)"

.PHONY: test
test: ## Run tests
	@echo -e "$(GREEN)Running tests...$(NC)"
	uv run pytest tests/ -v --cov=src --cov-report=html
	@echo -e "$(GREEN)Tests completed!$(NC)"

.PHONY: lint
lint: ## Run linting
	@echo -e "$(GREEN)Running linting...$(NC)"
	uv run flake8 src/ training/
	uv run pylint src/ training/ || true
	@echo -e "$(GREEN)Linting completed!$(NC)"

.PHONY: format
format: ## Format code
	@echo -e "$(GREEN)Formatting code...$(NC)"
	uv run black src/ training/
	uv run isort src/ training/
	@echo -e "$(GREEN)Code formatted!$(NC)"

.PHONY: typecheck
typecheck: ## Run type checking
	@echo -e "$(GREEN)Running type checking...$(NC)"
	uv run mypy src/ training/
	@echo -e "$(GREEN)Type checking completed!$(NC)"

.PHONY: generate-sample-data
generate-sample-data: ## Generate sample training data
	@echo -e "$(GREEN)Generating sample training data...$(NC)"
	uv run python -c "from training.data_loader import JSONTrainingDataGenerator; g = JSONTrainingDataGenerator(); g.generate_sample_training_data('sample_training_data.json', 1000)"
	@echo -e "$(GREEN)Sample data generated: sample_training_data.json$(NC)"

# Docker commands
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo -e "$(GREEN)Building Docker image...$(NC)"
	DOCKER_BUILDKIT=1 docker build --target production -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo -e "$(GREEN)Docker image built: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

.PHONY: docker-run
docker-run: docker-build ## Run Docker container locally
	@echo -e "$(GREEN)Starting Docker container...$(NC)"
	@# Stop and remove existing container if it exists
	@docker stop ml-risk-service 2>/dev/null || true
	@docker rm ml-risk-service 2>/dev/null || true
	@# Ensure directories exist for storage
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
		-e LOG_LEVEL="${LOG_LEVEL:-info}" \
		-e GRPC_PORT=8080 \
		-e HEALTH_PORT=8081 \
		-e REST_PORT=8090 \
		-e ENABLE_REST=true \
		-e TRAINING_CENTRAL_API_TOKEN \
		-e TRAINING_CENTRAL_ENDPOINT \
		-e PREDICTION_CENTRAL_API_TOKEN \
		-e PREDICTION_CENTRAL_ENDPOINT \
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

wait-for-ready:
	@echo -e "$(BLUE)Waiting for service to be ready...$(NC)"
	@# Wait for service to be ready (max 60 seconds)
	@MAX_WAIT=60; \
	ELAPSED=0; \
	while [ $$ELAPSED -lt $$MAX_WAIT ]; do \
		if curl -s -f http://localhost:8090/api/v1/health > /dev/null 2>&1; then \
			echo -e "$(GREEN)✓ Service is ready (waited $$ELAPSED seconds)$(NC)"; \
			break; \
		fi; \
		sleep 1; \
		ELAPSED=$$((ELAPSED + 1)); \
		if [ $$ELAPSED -eq $$MAX_WAIT ]; then \
			echo -e "$(RED)Service failed to start within $$MAX_WAIT seconds$(NC)"; \
			docker logs ml-risk-service; \
			exit 1; \
		fi; \
	done

.PHONY: train-predict-central
train-predict-central: docker-run wait-for-ready ## Train model with Central data, reload it, and validate predictions
	@echo -e "$(BLUE)Training the model with Central data"
	curl -X POST http://localhost:8090/api/v1/training/central/train-full | jq .
	@echo -e "$(BLUE)Reloading the model"
	curl -X POST http://localhost:8090/api/v1/models/stackrox-risk-model/reload | jq .
	@echo -e "$(BLUE)Validating the predictions"
	curl -X POST http://localhost:8090/api/v1/training/central/validate-predictions | jq .

.PHONY: test-prediction
test-prediction: ## Test prediction endpoint via REST API
	@echo -e "$(BLUE)Testing prediction endpoint...$(NC)"
	@echo -e "$(BLUE)Checking model health...$(NC)"
	curl -s http://localhost:8090/api/v1/models/stackrox-risk-model/health | python3 -m json.tool
	@echo -e "$(GREEN)Prediction endpoint test completed!$(NC)"

.PHONY: test-train-predict-central
test-train-predict-central: docker-run ## Train model with Central data, reload it, and validate predictions
	@echo -e "$(GREEN)Starting full Central training, reload, and validation workflow...$(NC)"
	@echo -e "$(BLUE)Waiting for service to be ready...$(NC)"
	@# Wait for service to be ready (max 60 seconds)
	@MAX_WAIT=60; \
	ELAPSED=0; \
	while [ $$ELAPSED -lt $$MAX_WAIT ]; do \
		if curl -s -f http://localhost:8090/api/v1/health > /dev/null 2>&1; then \
			echo -e "$(GREEN)✓ Service is ready (waited $$ELAPSED seconds)$(NC)"; \
			break; \
		fi; \
		sleep 1; \
		ELAPSED=$$((ELAPSED + 1)); \
		if [ $$ELAPSED -eq $$MAX_WAIT ]; then \
			echo -e "$(RED)Service failed to start within $$MAX_WAIT seconds$(NC)"; \
			docker logs ml-risk-service; \
			exit 1; \
		fi; \
	done && \
	echo -e "$(BLUE)Pre-flight: Checking Central API configuration...$(NC)" && \
	if [ -z "$$TRAINING_CENTRAL_API_TOKEN" ]; then \
		echo -e "$(YELLOW)Warning: TRAINING_CENTRAL_API_TOKEN environment variable is not set$(NC)"; \
		echo -e "$(YELLOW)The Central API may use default configuration or fail$(NC)"; \
	else \
		echo -e "$(GREEN)✓ TRAINING_CENTRAL_API_TOKEN is configured$(NC)"; \
	fi && \
	echo -e "$(BLUE)Testing Central API connection...$(NC)" && \
	CONNECTION_RESPONSE=$$(curl -s -X POST "http://localhost:8090/api/v1/training/central/test-connection" \
		-H "Content-Type: application/json" \
		-w "\n%{http_code}") && \
	CONNECTION_HTTP_CODE=$$(echo "$$CONNECTION_RESPONSE" | tail -n 1) && \
	CONNECTION_BODY=$$(echo "$$CONNECTION_RESPONSE" | sed '$$d') && \
	if [ "$$CONNECTION_HTTP_CODE" != "200" ]; then \
		echo -e "$(RED)Central API connection test failed with HTTP $$CONNECTION_HTTP_CODE$(NC)"; \
		echo "$$CONNECTION_BODY" | python3 -m json.tool 2>/dev/null || echo "$$CONNECTION_BODY"; \
		echo -e "$(YELLOW)Configuration help:$(NC)"; \
		echo -e "$(YELLOW)  1. Set TRAINING_CENTRAL_API_TOKEN environment variable$(NC)"; \
		echo -e "$(YELLOW)  2. Ensure Central endpoint is accessible$(NC)"; \
		echo -e "$(YELLOW)  3. Check src/config/feature_config.yaml settings$(NC)"; \
		exit 1; \
	fi && \
	CONNECTION_SUCCESS=$$(echo "$$CONNECTION_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('success', False))" 2>/dev/null || echo "false") && \
	if [ "$$CONNECTION_SUCCESS" != "True" ]; then \
		echo -e "$(RED)Central API connection test reported failure$(NC)"; \
		echo "$$CONNECTION_BODY" | python3 -m json.tool 2>/dev/null || echo "$$CONNECTION_BODY"; \
		CONNECTION_MESSAGE=$$(echo "$$CONNECTION_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('message', 'Unknown error'))" 2>/dev/null || echo "Unknown error"); \
		echo -e "$(RED)Error: $$CONNECTION_MESSAGE$(NC)"; \
		echo -e "$(YELLOW)Configuration help:$(NC)"; \
		echo -e "$(YELLOW)  1. Set TRAINING_CENTRAL_API_TOKEN: export TRAINING_CENTRAL_API_TOKEN='your-token'$(NC)"; \
		echo -e "$(YELLOW)  2. Verify Central endpoint in src/config/feature_config.yaml$(NC)"; \
		echo -e "$(YELLOW)  3. Check authentication and network connectivity$(NC)"; \
		exit 1; \
	fi && \
	echo -e "$(GREEN)✓ Central API connection successful$(NC)" && \
	echo -e "$(BLUE)Step 1: Training model with Central data...$(NC)" && \
	TRAIN_RESPONSE=$$(curl -s -X POST "http://localhost:8090/api/v1/training/central/train-full?test_mode=true" \
		-H "Content-Type: application/json" \
		-w "\n%{http_code}") && \
	TRAIN_HTTP_CODE=$$(echo "$$TRAIN_RESPONSE" | tail -n 1) && \
	TRAIN_BODY=$$(echo "$$TRAIN_RESPONSE" | sed '$$d') && \
	if [ "$$TRAIN_HTTP_CODE" != "200" ]; then \
		echo -e "$(RED)Training failed with HTTP $$TRAIN_HTTP_CODE$(NC)"; \
		echo "$$TRAIN_BODY" | python3 -m json.tool 2>/dev/null || echo "$$TRAIN_BODY"; \
		exit 1; \
	fi && \
	TRAIN_SUCCESS=$$(echo "$$TRAIN_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('success', False))" 2>/dev/null || echo "false") && \
	if [ "$$TRAIN_SUCCESS" != "True" ]; then \
		echo -e "$(RED)Training reported failure$(NC)"; \
		echo "$$TRAIN_BODY" | python3 -m json.tool 2>/dev/null || echo "$$TRAIN_BODY"; \
		exit 1; \
	fi && \
	MODEL_VERSION=$$(echo "$$TRAIN_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('model_version', ''))" 2>/dev/null || echo "") && \
	echo -e "$(GREEN)✓ Training completed successfully! Model version: $$MODEL_VERSION$(NC)" && \
	echo -e "$(BLUE)Step 2: Confirming model exists in API...$(NC)" && \
	MODELS_RESPONSE=$$(curl -s -X GET "http://localhost:8090/api/v1/models" \
		-H "Content-Type: application/json" \
		-w "\n%{http_code}") && \
	MODELS_HTTP_CODE=$$(echo "$$MODELS_RESPONSE" | tail -n 1) && \
	MODELS_BODY=$$(echo "$$MODELS_RESPONSE" | sed '$$d') && \
	if [ "$$MODELS_HTTP_CODE" != "200" ]; then \
		echo -e "$(RED)Failed to list models with HTTP $$MODELS_HTTP_CODE$(NC)"; \
		echo "$$MODELS_BODY" | python3 -m json.tool 2>/dev/null || echo "$$MODELS_BODY"; \
		exit 1; \
	fi && \
	MODEL_COUNT=$$(echo "$$MODELS_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(len(data.get('models', [])))" 2>/dev/null || echo "0") && \
	if [ "$$MODEL_COUNT" = "0" ]; then \
		echo -e "$(RED)No models found in API$(NC)"; \
		echo "$$MODELS_BODY" | python3 -m json.tool 2>/dev/null || echo "$$MODELS_BODY"; \
		exit 1; \
	fi && \
	echo -e "$(GREEN)✓ Model confirmed in API ($$MODEL_COUNT models found)$(NC)" && \
	echo -e "$(BLUE)Step 3: Reloading model...$(NC)" && \
	RELOAD_RESPONSE=$$(curl -s -X POST "http://localhost:8090/api/v1/models/stackrox-risk-model/reload" \
		-H "Content-Type: application/json" \
		-d '{"force_reload": true}' \
		-w "\n%{http_code}") && \
	RELOAD_HTTP_CODE=$$(echo "$$RELOAD_RESPONSE" | tail -n 1) && \
	RELOAD_BODY=$$(echo "$$RELOAD_RESPONSE" | sed '$$d') && \
	if [ "$$RELOAD_HTTP_CODE" != "200" ]; then \
		echo -e "$(RED)Model reload failed with HTTP $$RELOAD_HTTP_CODE$(NC)"; \
		echo "$$RELOAD_BODY" | python3 -m json.tool 2>/dev/null || echo "$$RELOAD_BODY"; \
		exit 1; \
	fi && \
	RELOAD_SUCCESS=$$(echo "$$RELOAD_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('success', False))" 2>/dev/null || echo "false") && \
	if [ "$$RELOAD_SUCCESS" != "True" ]; then \
		echo -e "$(RED)Model reload reported failure$(NC)"; \
		echo "$$RELOAD_BODY" | python3 -m json.tool 2>/dev/null || echo "$$RELOAD_BODY"; \
		exit 1; \
	fi && \
	echo -e "$(GREEN)✓ Model reload completed successfully!$(NC)" && \
	echo -e "$(BLUE)Step 4: Validating predictions with prediction Central...$(NC)" && \
	if [ -z "$$PREDICTION_CENTRAL_API_TOKEN" ]; then \
		echo -e "$(YELLOW)⚠ PREDICTION_CENTRAL_API_TOKEN not set - skipping validation step$(NC)"; \
		echo -e "$(YELLOW)To enable validation, set PREDICTION_CENTRAL_API_TOKEN and PREDICTION_CENTRAL_ENDPOINT$(NC)"; \
	else \
		echo -e "$(GREEN)✓ PREDICTION_CENTRAL_API_TOKEN is configured$(NC)" && \
		VALIDATE_RESPONSE=$$(curl -s -X POST "http://localhost:8090/api/v1/training/central/validate-predictions?limit=10" \
			-H "Content-Type: application/json" \
			-w "\n%{http_code}") && \
		VALIDATE_HTTP_CODE=$$(echo "$$VALIDATE_RESPONSE" | tail -n 1) && \
		VALIDATE_BODY=$$(echo "$$VALIDATE_RESPONSE" | sed '$$d') && \
		if [ "$$VALIDATE_HTTP_CODE" != "200" ]; then \
			echo -e "$(RED)Validation failed with HTTP $$VALIDATE_HTTP_CODE$(NC)"; \
			echo "$$VALIDATE_BODY" | python3 -m json.tool 2>/dev/null || echo "$$VALIDATE_BODY"; \
			echo -e "$(YELLOW)Continuing without validation...$(NC)"; \
		else \
			VALIDATE_SUCCESS=$$(echo "$$VALIDATE_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('success', False))" 2>/dev/null || echo "false") && \
			if [ "$$VALIDATE_SUCCESS" != "True" ]; then \
				echo -e "$(RED)Validation reported failure$(NC)"; \
				echo "$$VALIDATE_BODY" | python3 -m json.tool 2>/dev/null || echo "$$VALIDATE_BODY"; \
				echo -e "$(YELLOW)Continuing without validation...$(NC)"; \
			else \
				TOTAL_SAMPLES=$$(echo "$$VALIDATE_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('validation_summary', {}).get('total_samples', 0))" 2>/dev/null || echo "0") && \
				MAE=$$(echo "$$VALIDATE_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('metrics', {}).get('mae', 0))" 2>/dev/null || echo "0") && \
				CORRELATION=$$(echo "$$VALIDATE_BODY" | python3 -c "import sys, json; data=json.load(sys.stdin); print(data.get('metrics', {}).get('correlation', 0))" 2>/dev/null || echo "0") && \
				echo -e "$(GREEN)✓ Validation completed successfully!$(NC)" && \
				echo -e "$(GREEN)  Total samples: $$TOTAL_SAMPLES$(NC)" && \
				echo -e "$(GREEN)  MAE: $$MAE$(NC)" && \
				echo -e "$(GREEN)  Correlation: $$CORRELATION$(NC)"; \
			fi; \
		fi; \
	fi && \
	echo -e "$(GREEN)🎉 Full Central training, reload, and validation workflow completed successfully!$(NC)"

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

# Model analysis commands
.PHONY: analyze-n-estimators
analyze-n-estimators: docker-build ## Analyze RandomForest convergence for different n_estimators values
	@echo -e "$(GREEN)Running n_estimators convergence analysis in Docker...$(NC)"
	docker run --rm \
		--entrypoint python \
		-e TRAINING_CENTRAL_API_TOKEN \
		-e TRAINING_CENTRAL_ENDPOINT \
		$(DOCKER_IMAGE):$(DOCKER_TAG) \
		scripts/analyze_n_estimators_convergence.py \
			--data-source central \
			--limit 1000
	@echo -e "$(GREEN)Analysis complete!$(NC)"

.PHONY: analyze-n-estimators-synthetic
analyze-n-estimators-synthetic: docker-build ## Run convergence analysis with synthetic data (no Central API needed)
	@echo -e "$(GREEN)Running n_estimators convergence analysis in Docker (synthetic data)...$(NC)"
	@mkdir -p $(PWD)/reports
	docker run --rm \
		--entrypoint python \
		-v $(PWD)/reports:/app/reports \
		$(DOCKER_IMAGE):$(DOCKER_TAG) \
		scripts/analyze_n_estimators_convergence.py \
			--data-source synthetic \
			--limit 1000 \
			--output-dir /app/reports
	@echo -e "$(GREEN)Analysis complete!$(NC)"
	@echo -e "$(BLUE)  Plot: reports/n_estimators_convergence.png$(NC)"
	@echo -e "$(BLUE)  Data: reports/n_estimators_results.json$(NC)"

# Cleanup commands
.PHONY: clean
clean: ## Clean up temporary files, trained models, and training reports
	@echo -e "$(YELLOW)Cleaning up temporary files...$(NC)"
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete
	find . -type d -name "*.egg-info" -exec rm -rf {} + 2>/dev/null || true
	rm -rf .coverage htmlcov/ .pytest_cache/
	@echo -e "$(YELLOW)Cleaning up models and reports...$(NC)"
	rm -rf ./models/ ./deploy/models/
	rm -rf ./reports/ ./deploy/reports/
	rm -f sample_training_data.json
	rm -f *.pkl
	@echo -e "$(GREEN)Cleanup completed!$(NC)"

.PHONY: clean-docker
clean-docker: ## Clean up Docker resources
	@echo -e "$(YELLOW)Cleaning up Docker resources...$(NC)"
	docker-compose down --volumes --remove-orphans || true
	docker system prune -f
	@echo -e "$(GREEN)Docker cleanup completed!$(NC)"

# Development workflow
.PHONY: dev-setup
dev-setup: setup-dev generate-sample-data ## Complete development setup
	@echo -e "$(GREEN)Development setup completed!$(NC)"
	@echo "Next steps:"
	@echo "  1. Run 'make train-model' to train a sample model"
	@echo "  2. Run 'make docker-build && make docker-run' to test locally"
	@echo "  3. Run 'make k8s-deploy' to deploy to Kubernetes"

.PHONY: ci-pipeline
ci-pipeline: setup-dev lint typecheck test docker-build ## Run CI pipeline
	@echo -e "$(GREEN)CI pipeline completed successfully!$(NC)"

.PHONY: check
check: lint typecheck test ## Run all checks (lint, typecheck, test)
	@echo -e "$(GREEN)All checks passed!$(NC)"

# Production deployment
.PHONY: deploy-prod
deploy-prod: docker-build docker-push k8s-deploy ## Deploy to production
	@echo -e "$(GREEN)Production deployment completed!$(NC)"

# Default target
.DEFAULT_GOAL := help
