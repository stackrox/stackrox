# Multi-stage Dockerfile for ML Risk Service with UV optimizations
FROM python:3.11-slim AS builder

# Set environment variables for UV optimization
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    UV_LINK_MODE=copy \
    UV_COMPILE_BYTECODE=1 \
    UV_PYTHON_CACHE_DIR=/root/.cache/python

# Install system dependencies needed for building
RUN apt-get update && apt-get install -y \
    gcc \
    g++ \
    libc6-dev \
    libgomp1 \
    curl \
    && rm -rf /var/lib/apt/lists/*

# Install specific UV version for reproducibility
ENV UV_VERSION=0.4.18
RUN curl -LsSf https://astral.sh/uv/${UV_VERSION}/install.sh | sh && \
    /root/.cargo/bin/uv --version
ENV PATH="/root/.cargo/bin:$PATH"

# Create app directory
WORKDIR /app

# Copy only dependency files first for better layer caching
COPY pyproject.toml uv.lock ./

# Install dependencies first (separate from project code for better caching)
RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=cache,target=/root/.cache/python \
    uv sync --frozen --no-dev --extra ml --no-install-project --no-editable

# Copy source code after dependencies are cached
COPY src/ ./src/
COPY README.md ./

# Install the project itself with bytecode compilation
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev --extra ml --no-editable --compile-bytecode

# Development stage (includes dev dependencies and editable install)
FROM builder AS development

# Install with dev dependencies for development
RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=cache,target=/root/.cache/python \
    uv sync --frozen --extra dev --extra ml

# Set up development environment
WORKDIR /app
ENV PYTHONPATH=/app/src
EXPOSE 8081 8090

# Development entrypoint
CMD ["uv", "run", "python", "-m", "src.api.rest_server", "--host", "0.0.0.0", "--port", "8090", "--reload"]

# Production stage
FROM python:3.11-slim AS production

# Set production environment variables (no UV cache needed in production)
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PYTHONPATH=/app

# Install runtime dependencies and create user
RUN apt-get update && apt-get install -y \
    libgomp1 \
    curl \
    && rm -rf /var/lib/apt/lists/* \
    && groupadd -r mlrisk \
    && useradd -r -g mlrisk -s /bin/false mlrisk

# Copy optimized virtual environment from builder
COPY --from=builder /app/.venv /app/.venv

# Add virtual environment to PATH
ENV PATH="/app/.venv/bin:$PATH"

# Create app directory structure
WORKDIR /app
RUN mkdir -p /app/models /app/reports /app/logs \
    && chown -R mlrisk:mlrisk /app

# Copy application code from builder (where it was installed as a package)
COPY --from=builder /app/src/ /app/src/

# Copy configuration files
# Use template config with environment variable placeholders
COPY src/config/feature_config.yaml /app/config/feature_config.yaml

# Copy scripts
COPY scripts/entrypoint.sh /app/entrypoint.sh
COPY scripts/healthcheck.sh /app/healthcheck.sh

# Set script permissions
RUN chmod +x /app/entrypoint.sh /app/healthcheck.sh && \
    chown mlrisk:mlrisk /app/entrypoint.sh /app/healthcheck.sh

# Switch to non-root user
USER mlrisk

# Expose ports
EXPOSE 8081 8090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=60s --retries=3 \
    CMD /app/healthcheck.sh

# Default entrypoint
ENTRYPOINT ["/app/entrypoint.sh"]

# Labels
LABEL maintainer="StackRox ML Team" \
      description="ML Risk Ranking Service for StackRox" \
      version="1.0.0"