import React, { useState } from 'react';
import {
    TextInput,
    Spinner,
    FormGroup,
    FormHelperText,
    HelperText,
    HelperTextItem,
    Flex,
    FlexItem,
} from '@patternfly/react-core';

import type { NaturalLanguageSearchInputProps } from './types';
import { createOllamaProvider } from '../../services/aiProviders/ollamaProvider';
import { createAISearchParserService } from '../../services/aiSearchParserService';

const DEFAULT_PLACEHOLDER = 'Search using natural language (e.g., "critical CVEs in production")';
const DEFAULT_MAX_LENGTH = 500;

/**
 * NaturalLanguageSearchInput
 * A text input component that converts natural language queries into structured search filters using AI
 */
function NaturalLanguageSearchInput({
    searchFilterConfig,
    onFilterGenerated,
    onError,
    placeholder = DEFAULT_PLACEHOLDER,
    maxLength = DEFAULT_MAX_LENGTH,
    isDisabled = false,
}: NaturalLanguageSearchInputProps) {
    const [query, setQuery] = useState('');
    const [isLoading, setIsLoading] = useState(false);

    /**
     * Handle input change
     */
    const handleInputChange = (
        _event: React.FormEvent<HTMLInputElement>,
        value: string
    ) => {
        // Enforce max length
        if (value.length <= maxLength) {
            setQuery(value);
        }
    };

    /**
     * Handle search submission
     */
    const handleSearch = async () => {
        if (!query.trim() || isLoading) {
            return;
        }

        setIsLoading(true);

        try {
            // Create AI provider and parser service
            const provider = createOllamaProvider();
            const parserService = createAISearchParserService({ provider });

            // Parse the natural language query
            const result = await parserService.parseNaturalLanguageQuery(
                query,
                searchFilterConfig
            );

            // Call the success callback
            onFilterGenerated(result);

            // Clear the input after successful search
            setQuery('');
        } catch (error) {
            // Handle errors
            if (onError) {
                // Type guard to check if error has expected shape
                if (
                    error &&
                    typeof error === 'object' &&
                    'message' in error &&
                    'type' in error &&
                    'originalQuery' in error
                ) {
                    onError(error as any); // Safe cast since we checked the structure
                } else {
                    // Fallback error
                    onError({
                        message:
                            error instanceof Error ? error.message : 'Unknown error occurred',
                        type: 'unknown',
                        originalQuery: query,
                    });
                }
            }
        } finally {
            setIsLoading(false);
        }
    };

    /**
     * Handle Enter key press
     */
    const handleKeyPress = (event: React.KeyboardEvent<HTMLInputElement>) => {
        if (event.key === 'Enter') {
            event.preventDefault();
            handleSearch();
        }
    };

    return (
        <FormGroup fieldId="natural-language-search" label="Natural Language Search">
            <Flex spaceItems={{ default: 'spaceItemsSm' }} alignItems={{ default: 'alignItemsCenter' }}>
                <FlexItem flex={{ default: 'flex_1' }}>
                    <TextInput
                        id="natural-language-search"
                        type="text"
                        value={query}
                        onChange={handleInputChange}
                        onKeyPress={handleKeyPress}
                        placeholder={placeholder}
                        isDisabled={isDisabled || isLoading}
                        aria-label="Natural language search input"
                    />
                </FlexItem>
                {isLoading && (
                    <FlexItem>
                        <Spinner size="md" aria-label="Processing query" />
                    </FlexItem>
                )}
            </Flex>
            <FormHelperText>
                <HelperText>
                    <HelperTextItem>
                        Enter a natural language query and press Enter to search (max {maxLength}{' '}
                        characters)
                    </HelperTextItem>
                </HelperText>
            </FormHelperText>
        </FormGroup>
    );
}

export default NaturalLanguageSearchInput;
