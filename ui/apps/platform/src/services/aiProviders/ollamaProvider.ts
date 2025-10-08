/**
 * Ollama AI Provider implementation
 * Provides local AI completions using Ollama REST API
 */

import { AIProvider, AIResponse, AIProviderConfig } from './types';

/**
 * Ollama API request format
 */
type OllamaGenerateRequest = {
    model: string;
    prompt: string;
    stream: boolean;
    options?: {
        temperature?: number;
        top_p?: number;
    };
};

/**
 * Ollama API response format
 */
type OllamaGenerateResponse = {
    model: string;
    created_at: string;
    response: string;
    done: boolean;
    context?: number[];
    total_duration?: number;
    load_duration?: number;
    prompt_eval_count?: number;
    prompt_eval_duration?: number;
    eval_count?: number;
    eval_duration?: number;
};

/**
 * Default Ollama configuration
 */
const DEFAULT_CONFIG = {
    url: 'http://localhost:11434',
    model: 'gemma3:4b',
    timeout: 30000, // 30 seconds - gemma3:4b typically takes 10-24s
};

/**
 * OllamaProvider implements the AIProvider interface for local Ollama instances
 */
export class OllamaProvider implements AIProvider {
    private config: Required<Pick<AIProviderConfig, 'url' | 'model' | 'timeout'>>;

    constructor(config?: Partial<AIProviderConfig>) {
        this.config = {
            url: config?.url || DEFAULT_CONFIG.url,
            model: config?.model || DEFAULT_CONFIG.model,
            timeout: config?.timeout || DEFAULT_CONFIG.timeout,
        };
    }

    getName(): string {
        return 'Ollama';
    }

    /**
     * Check if Ollama is available by hitting the tags endpoint
     */
    async isAvailable(): Promise<boolean> {
        try {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 2000); // 2 second timeout for health check

            const response = await fetch(`${this.config.url}/api/tags`, {
                signal: controller.signal,
            });

            clearTimeout(timeoutId);
            return response.ok;
        } catch (error) {
            // Connection failed, timeout, or network error
            return false;
        }
    }

    /**
     * Generate a completion using Ollama's generate API
     */
    async generateCompletion(prompt: string): Promise<AIResponse> {
        if (!prompt || prompt.trim().length === 0) {
            throw new Error('Prompt cannot be empty');
        }

        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), this.config.timeout);

        try {
            const requestBody: OllamaGenerateRequest = {
                model: this.config.model,
                prompt,
                stream: false,
                options: {
                    temperature: 0.7,
                    top_p: 0.9,
                },
            };

            const response = await fetch(`${this.config.url}/api/generate`, {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestBody),
                signal: controller.signal,
            });

            clearTimeout(timeoutId);

            if (!response.ok) {
                const errorText = await response.text().catch(() => 'Unknown error');
                throw new Error(
                    `Ollama API error (${response.status}): ${errorText}`
                );
            }

            const data: OllamaGenerateResponse = await response.json();

            // Log the full response for debugging
            console.log('🤖 Ollama API Response:', {
                model: data.model,
                response: data.response,
                done: data.done,
                total_duration_ms: data.total_duration ? Math.round(data.total_duration / 1000000) : null,
                eval_count: data.eval_count,
            });

            if (!data.response) {
                throw new Error('Ollama response missing content');
            }

            return {
                content: data.response.trim(),
                // Ollama doesn't provide confidence scores, so we don't set it
                // The parser service will need to calculate confidence based on output quality
            };
        } catch (error) {
            clearTimeout(timeoutId);

            if (error instanceof Error) {
                // Handle abort/timeout
                if (error.name === 'AbortError') {
                    throw new Error(
                        `Ollama request timed out after ${this.config.timeout}ms`
                    );
                }

                // Handle network errors
                if (error.message.includes('fetch')) {
                    throw new Error(
                        `Cannot connect to Ollama at ${this.config.url}. Is Ollama running?`
                    );
                }

                // Re-throw other errors
                throw error;
            }

            // Fallback for unknown error types
            throw new Error('Unknown error occurred while calling Ollama');
        }
    }
}

/**
 * Create and return a configured Ollama provider instance
 */
export function createOllamaProvider(config?: Partial<AIProviderConfig>): OllamaProvider {
    return new OllamaProvider(config);
}
