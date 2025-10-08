/**
 * AI Provider types for natural language search
 */

/**
 * Response from an AI provider
 */
export type AIResponse = {
    /** The generated content/completion from the AI */
    content: string;
    /** Confidence score from 0-1 indicating how confident the AI is in its response */
    confidence?: number;
    /** Optional reasoning or explanation for the response */
    reasoning?: string;
};

/**
 * Configuration for AI providers
 */
export type AIProviderConfig = {
    /** Provider name (ollama, anthropic, openai) */
    provider: 'ollama' | 'anthropic' | 'openai';
    /** API endpoint URL */
    url?: string;
    /** Model name to use */
    model?: string;
    /** API key for cloud providers */
    apiKey?: string;
    /** Request timeout in milliseconds */
    timeout?: number;
};

/**
 * Interface that all AI providers must implement
 */
export interface AIProvider {
    /**
     * Generate a completion for the given prompt
     * @param prompt The prompt to send to the AI
     * @returns Promise resolving to AI response
     */
    generateCompletion(prompt: string): Promise<AIResponse>;

    /**
     * Check if the provider is available/reachable
     * @returns Promise resolving to true if available
     */
    isAvailable(): Promise<boolean>;

    /**
     * Get the name of this provider
     * @returns Provider name
     */
    getName(): string;
}
