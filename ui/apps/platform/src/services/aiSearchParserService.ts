/**
 * AI Search Parser Service
 * Converts natural language queries into structured search filters using AI
 */

import type { CompoundSearchFilterConfig } from 'Components/CompoundSearchFilter/types';
import type { ParseResult, ParseError } from 'Components/NaturalLanguageSearch/types';
import type { SearchFilter } from 'types/search';
import type { AIProvider } from './aiProviders/types';
import { sanitizeInput, InputValidationError } from './inputSanitizer';
import { buildFilterSchema, formatSchemaForPrompt, getSchemaExamples } from './filterSchemaBuilder';

/**
 * Configuration for the AI search parser
 */
export type AISearchParserConfig = {
    /** AI provider to use for completions */
    provider: AIProvider;
    /** Minimum confidence threshold (0-1) */
    minConfidence?: number;
    /** Maximum query length */
    maxQueryLength?: number;
};

/**
 * Default configuration
 */
const DEFAULT_CONFIG = {
    minConfidence: 0.5,
    maxQueryLength: 500,
};

/**
 * Build the AI prompt for parsing natural language queries
 */
function buildPrompt(query: string, filterConfig: CompoundSearchFilterConfig): string {
    const schema = buildFilterSchema(filterConfig);
    const schemaExamples = getSchemaExamples(schema);

    return `You are a search query parser that converts natural language into structured search filters.

Your task: Convert the user's natural language query into a JSON search filter object.

${schemaExamples}

IMPORTANT RULES:
1. ONLY use the search terms provided in the schema above
2. Output MUST be valid JSON matching this exact format:
{
  "searchFilter": {
    "Search Term": "value",
    "Another Search Term": "value2"
  },
  "confidence": 0.9,
  "reasoning": "Brief explanation of the interpretation"
}

3. Confidence scoring (0.0 to 1.0):
   - 0.9-1.0: Very confident, clear unambiguous query
   - 0.7-0.8: Confident, minor assumptions made
   - 0.5-0.6: Moderate confidence, some ambiguity
   - 0.0-0.4: Low confidence, very ambiguous

4. For date-picker fields, use ISO 8601 format (YYYY-MM-DD)
5. For condition-number fields, use operators like ">5", "<10", ">=3.0"
6. For select fields, use exact option values from the schema
7. Values can be strings or arrays of strings for multiple selections
8. If query is completely unclear, set confidence to 0.3 or lower

EXAMPLES:

Query: "critical CVEs"
{
  "searchFilter": {
    "Severity": "CRITICAL_VULNERABILITY_SEVERITY"
  },
  "confidence": 0.95,
  "reasoning": "Clear request for critical severity CVEs"
}

Query: "fixable critical CVEs in production from last week"
{
  "searchFilter": {
    "Severity": "CRITICAL_VULNERABILITY_SEVERITY",
    "Fixable": "true",
    "Deployment": "*prod*"
  },
  "confidence": 0.75,
  "reasoning": "Interpreted 'production' as deployment name containing 'prod'. 'Last week' would need specific date filter which requires more context."
}

USER QUERY: "${query}"

OUTPUT (JSON only, no explanation):`;
}

/**
 * Parse the AI response and extract search filter data
 */
function parseAIResponse(aiContent: string): {
    searchFilter: SearchFilter;
    confidence: number;
    reasoning?: string;
} {
    // Try to extract JSON from the response (AI might wrap it in code blocks)
    let jsonContent = aiContent.trim();

    // Remove markdown code blocks if present
    if (jsonContent.startsWith('```json')) {
        jsonContent = jsonContent.replace(/```json\n?/, '').replace(/\n?```\s*$/, '');
    } else if (jsonContent.startsWith('```')) {
        jsonContent = jsonContent.replace(/```\n?/, '').replace(/\n?```\s*$/, '');
    }

    // Parse the JSON
    const parsed = JSON.parse(jsonContent);

    // Validate structure
    if (!parsed.searchFilter || typeof parsed.searchFilter !== 'object') {
        throw new Error('AI response missing searchFilter object');
    }

    if (typeof parsed.confidence !== 'number') {
        throw new Error('AI response missing confidence score');
    }

    // Ensure confidence is between 0 and 1
    const confidence = Math.max(0, Math.min(1, parsed.confidence));

    return {
        searchFilter: parsed.searchFilter as SearchFilter,
        confidence,
        reasoning: parsed.reasoning,
    };
}

/**
 * AI Search Parser Service
 * Converts natural language queries into structured search filters
 */
export class AISearchParserService {
    private config: Required<AISearchParserConfig>;

    constructor(config: AISearchParserConfig) {
        this.config = {
            ...DEFAULT_CONFIG,
            ...config,
        };
    }

    /**
     * Parse a natural language query into a search filter
     *
     * @param query Natural language query from the user
     * @param filterConfig Available filter configuration
     * @returns Promise resolving to ParseResult
     * @throws ParseError if parsing fails
     */
    async parseNaturalLanguageQuery(
        query: string,
        filterConfig: CompoundSearchFilterConfig
    ): Promise<ParseResult> {
        try {
            // Step 1: Sanitize input
            let sanitizedQuery: string;
            try {
                sanitizedQuery = sanitizeInput(query, {
                    maxLength: this.config.maxQueryLength,
                });
            } catch (error) {
                if (error instanceof InputValidationError) {
                    throw this.createParseError(error.message, 'validation', query);
                }
                throw error;
            }

            // Step 2: Build prompt
            const prompt = buildPrompt(sanitizedQuery, filterConfig);

            // Step 3: Call AI provider
            let aiResponse;
            try {
                aiResponse = await this.config.provider.generateCompletion(prompt);
            } catch (error) {
                const message = error instanceof Error ? error.message : 'Unknown error';
                if (message.includes('timeout')) {
                    throw this.createParseError(
                        'AI request timed out. Please try again.',
                        'timeout',
                        query
                    );
                }
                throw this.createParseError(
                    `AI provider error: ${message}`,
                    'api',
                    query
                );
            }

            // Step 4: Parse AI response
            let parsedResponse;
            try {
                parsedResponse = parseAIResponse(aiResponse.content);
            } catch (error) {
                const message = error instanceof Error ? error.message : 'Invalid JSON';
                throw this.createParseError(
                    `Failed to parse AI response: ${message}`,
                    'api',
                    query
                );
            }

            // Step 5: Check confidence threshold
            if (parsedResponse.confidence < this.config.minConfidence) {
                // Don't throw error, but return low confidence result
                // The UI can decide how to handle it
            }

            // Step 6: Return result
            return {
                searchFilter: parsedResponse.searchFilter,
                confidence: parsedResponse.confidence,
                reasoning: parsedResponse.reasoning,
                originalQuery: query,
            };
        } catch (error) {
            // Re-throw ParseErrors
            if (this.isParseError(error)) {
                throw error;
            }

            // Wrap other errors
            const message = error instanceof Error ? error.message : 'Unknown error occurred';
            throw this.createParseError(message, 'unknown', query);
        }
    }

    /**
     * Check if the AI provider is available
     */
    async isProviderAvailable(): Promise<boolean> {
        return this.config.provider.isAvailable();
    }

    /**
     * Get the name of the AI provider being used
     */
    getProviderName(): string {
        return this.config.provider.getName();
    }

    /**
     * Helper to create ParseError objects
     */
    private createParseError(
        message: string,
        type: ParseError['type'],
        originalQuery: string
    ): ParseError {
        return {
            message,
            type,
            originalQuery,
        };
    }

    /**
     * Type guard for ParseError
     */
    private isParseError(error: unknown): error is ParseError {
        return (
            typeof error === 'object' &&
            error !== null &&
            'message' in error &&
            'type' in error &&
            'originalQuery' in error
        );
    }
}

/**
 * Factory function to create an AI search parser service
 */
export function createAISearchParserService(
    config: AISearchParserConfig
): AISearchParserService {
    return new AISearchParserService(config);
}
