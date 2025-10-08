/**
 * Types for Natural Language Search components
 */

import { SearchFilter } from 'types/search';

/**
 * Result of parsing a natural language query
 */
export type ParseResult = {
    /** The generated search filter from the natural language query */
    searchFilter: SearchFilter;
    /** Confidence score from 0-1 indicating parsing accuracy */
    confidence: number;
    /** Optional explanation of how the query was interpreted */
    reasoning?: string;
    /** Original natural language query */
    originalQuery: string;
};

/**
 * Error that occurred during query parsing
 */
export type ParseError = {
    /** Error message */
    message: string;
    /** Type of error */
    type: 'validation' | 'api' | 'timeout' | 'unknown';
    /** Original query that caused the error */
    originalQuery: string;
};

/**
 * Props for NaturalLanguageSearchInput component
 */
export type NaturalLanguageSearchInputProps = {
    /** Callback when a filter is successfully generated */
    onFilterGenerated: (result: ParseResult) => void;
    /** Callback when an error occurs */
    onError?: (error: ParseError) => void;
    /** Placeholder text for the input */
    placeholder?: string;
    /** Maximum query length in characters */
    maxLength?: number;
    /** Whether the component is disabled */
    isDisabled?: boolean;
};
