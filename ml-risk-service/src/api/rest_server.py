"""
FastAPI REST server for ML Risk Service with automatic OpenAPI documentation.
"""

import logging
import os
import time
from typing import Dict, Any, Optional
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI, Request, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from fastapi.middleware.trustedhost import TrustedHostMiddleware
from fastapi.responses import JSONResponse, RedirectResponse
from fastapi.openapi.docs import get_swagger_ui_html, get_redoc_html

from src.api.routers import risk, models, training, health
from src.api.schemas import ErrorResponse

logger = logging.getLogger(__name__)

# Global service start time
service_start_time = time.time()


@asynccontextmanager
async def lifespan(app: FastAPI):
    """
    Application lifespan context manager.
    Handles startup and shutdown events.
    """
    # Startup
    logger.info("ML Risk Service REST API starting up...")

    # Initialize services (could add model preloading here)
    logger.info("Services initialized")

    yield

    # Shutdown
    logger.info("ML Risk Service REST API shutting down...")


def create_app(config: Optional[Dict[str, Any]] = None) -> FastAPI:
    """
    Create and configure FastAPI application.

    Args:
        config: Optional configuration dictionary

    Returns:
        Configured FastAPI application
    """
    config = config or {}

    # Create FastAPI app
    app = FastAPI(
        title="StackRox ML Risk Service API",
        description="""
        REST API for StackRox ML Risk Ranking Service

        This API provides endpoints for:
        - **Risk Prediction**: Get risk scores for Kubernetes deployments
        - **Model Management**: Hot reload models and check health
        - **Training**: Train new models with custom data
        - **Monitoring**: Health checks and metrics for observability

        ## Authentication
        Currently no authentication is required. In production, this API should be
        secured with appropriate authentication and authorization mechanisms.

        ## Rate Limiting
        No rate limiting is currently implemented. Consider adding rate limiting
        for production deployments.

        ## Error Handling
        All endpoints return structured error responses with appropriate HTTP status codes.
        Check the response schemas for error details.
        """,
        version="1.0.0",
        docs_url="/docs",
        redoc_url="/redoc",
        openapi_url="/openapi.json",
        lifespan=lifespan
    )

    # Add CORS middleware
    app.add_middleware(
        CORSMiddleware,
        allow_origins=config.get("cors_origins", ["*"]),
        allow_credentials=True,
        allow_methods=["*"],
        allow_headers=["*"],
    )

    # Add trusted host middleware for security
    if config.get("trusted_hosts"):
        app.add_middleware(
            TrustedHostMiddleware,
            allowed_hosts=config["trusted_hosts"]
        )

    # Add request logging middleware
    @app.middleware("http")
    async def log_requests(request: Request, call_next):
        """Log HTTP requests for monitoring."""
        start_time = time.time()

        # Log request
        logger.info(f"Request: {request.method} {request.url}")

        try:
            response = await call_next(request)

            # Log response
            process_time = time.time() - start_time
            logger.info(
                f"Response: {response.status_code} "
                f"(took {process_time:.3f}s)"
            )

            # Add response headers
            response.headers["X-Process-Time"] = str(process_time)
            response.headers["X-API-Version"] = "1.0.0"

            return response

        except Exception as e:
            # Log errors
            process_time = time.time() - start_time
            logger.error(f"Request failed: {e} (took {process_time:.3f}s)")
            raise

    # Include routers
    app.include_router(risk.router, prefix="/api/v1")
    app.include_router(models.router, prefix="/api/v1")
    app.include_router(training.router, prefix="/api/v1")
    app.include_router(health.router, prefix="/api/v1")

    # Root endpoint
    @app.get("/", include_in_schema=False)
    async def root():
        """Redirect root to API documentation."""
        return RedirectResponse(url="/docs")

    # API info endpoint
    @app.get("/api/info", tags=["info"])
    async def api_info():
        """Get API information and service status."""
        uptime = time.time() - service_start_time

        return {
            "service": "StackRox ML Risk Service",
            "version": "1.0.0",
            "api_version": "v1",
            "description": "REST API for ML-based security risk assessment",
            "uptime_seconds": uptime,
            "documentation": {
                "swagger_ui": "/docs",
                "redoc": "/redoc",
                "openapi_spec": "/openapi.json"
            },
            "endpoints": {
                "risk_prediction": "/api/v1/prediction",
                "model_management": "/api/v1/models",
                "training": "/api/v1/training",
                "health": "/api/v1/health"
            }
        }

    # Custom OpenAPI schema with examples
    def custom_openapi():
        """Generate custom OpenAPI schema with enhanced documentation."""
        if app.openapi_schema:
            return app.openapi_schema

        from fastapi.openapi.utils import get_openapi

        openapi_schema = get_openapi(
            title=app.title,
            version=app.version,
            description=app.description,
            routes=app.routes,
        )

        # Add custom info
        openapi_schema["info"]["contact"] = {
            "name": "StackRox ML Team",
            "email": "ml-team@stackrox.com",
        }

        openapi_schema["info"]["license"] = {
            "name": "Proprietary",
        }

        # Add servers
        openapi_schema["servers"] = [
            {
                "url": "http://localhost:8090",
                "description": "Local development server"
            },
            {
                "url": "https://ml-risk-service.stackrox.com",
                "description": "Production server"
            }
        ]

        app.openapi_schema = openapi_schema
        return app.openapi_schema

    app.openapi = custom_openapi

    # Global exception handler
    @app.exception_handler(HTTPException)
    async def http_exception_handler(request: Request, exc: HTTPException):
        """Handle HTTP exceptions with structured error responses."""
        return JSONResponse(
            status_code=exc.status_code,
            content={
                "error": exc.detail,
                "status_code": exc.status_code,
                "timestamp": int(time.time()),
                "path": str(request.url)
            }
        )

    @app.exception_handler(Exception)
    async def general_exception_handler(request: Request, exc: Exception):
        """Handle unexpected exceptions."""
        logger.error(f"Unhandled exception: {exc}")
        return JSONResponse(
            status_code=500,
            content={
                "error": "Internal server error",
                "status_code": 500,
                "timestamp": int(time.time()),
                "path": str(request.url)
            }
        )

    return app


def main():
    """Main entry point for the REST API server."""
    import argparse
    import yaml

    parser = argparse.ArgumentParser(description='ML Risk Service REST API')
    parser.add_argument('--config', help='Configuration file path')
    parser.add_argument('--host', default='0.0.0.0', help='Host to bind to')
    parser.add_argument('--port', type=int, default=8090, help='Port to bind to')
    parser.add_argument('--workers', type=int, default=1, help='Number of worker processes')
    parser.add_argument('--reload', action='store_true', help='Enable auto-reload for development')
    parser.add_argument('--log-level', default='info', help='Log level')

    args = parser.parse_args()

    # Setup logging
    logging.basicConfig(
        level=getattr(logging, args.log_level.upper()),
        format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
    )

    # Load configuration
    config = {}
    if args.config and os.path.exists(args.config):
        with open(args.config, 'r') as f:
            config = yaml.safe_load(f)

    # Override with environment variables
    host = os.environ.get('REST_HOST', args.host)
    port = int(os.environ.get('REST_PORT', args.port))

    # Create app
    app = create_app(config)

    logger.info(f"Starting ML Risk Service REST API on {host}:{port}")
    logger.info(f"API documentation available at: http://{host}:{port}/docs")
    logger.info(f"OpenAPI spec available at: http://{host}:{port}/openapi.json")

    # Run server
    uvicorn.run(
        app,
        host=host,
        port=port,
        workers=args.workers if not args.reload else 1,
        reload=args.reload,
        log_level=args.log_level.lower(),  # uvicorn expects lowercase log levels
        access_log=True
    )


if __name__ == "__main__":
    main()