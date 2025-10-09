/**
 * Filter schema builder for AI natural language search
 * Extracts available filters from CompoundSearchFilterConfig and formats them for AI prompts
 */

import type {
    CompoundSearchFilterConfig,
    CompoundSearchFilterAttribute,
} from 'Components/CompoundSearchFilter/types';

/**
 * Schema for a single filter attribute
 */
export type FilterAttributeSchema = {
    /** Display name shown to users */
    displayName: string;
    /** Backend search term used in queries */
    searchTerm: string;
    /** Type of input (determines validation and format) */
    inputType: string;
    /** Available options for select/autocomplete types */
    options?: string[];
};

/**
 * Schema for a filter entity (e.g., CVE, Image, Deployment)
 */
export type FilterEntitySchema = {
    /** Display name of the entity */
    displayName: string;
    /** Available attributes that can be filtered */
    attributes: FilterAttributeSchema[];
};

/**
 * Complete filter schema for AI prompts
 */
export type FilterSchema = {
    /** List of entity schemas */
    entities: FilterEntitySchema[];
    /** Human-readable description of the schema */
    description: string;
};

/**
 * Convert a CompoundSearchFilterAttribute to FilterAttributeSchema
 */
function buildAttributeSchema(attribute: CompoundSearchFilterAttribute): FilterAttributeSchema {
    const schema: FilterAttributeSchema = {
        displayName: attribute.displayName,
        searchTerm: attribute.searchTerm,
        inputType: attribute.inputType,
    };

    // Extract options for select-type filters
    if (attribute.inputType === 'select' && 'inputProps' in attribute) {
        const inputProps = attribute.inputProps;

        if ('options' in inputProps) {
            // Simple options array
            schema.options = inputProps.options.map((opt) => opt.value);
        } else if ('groupOptions' in inputProps) {
            // Grouped options - flatten them
            schema.options = inputProps.groupOptions.flatMap((group) =>
                group.options.map((opt) => opt.value)
            );
        }
    }

    return schema;
}

/**
 * Build a filter schema from CompoundSearchFilterConfig
 *
 * @param config The compound search filter configuration
 * @param includeSpecialFilters Whether to include special filters like SEVERITY and FIXABLE
 * @returns FilterSchema object suitable for AI prompts
 */
export function buildFilterSchema(
    config: CompoundSearchFilterConfig,
    includeSpecialFilters: boolean = true
): FilterSchema {
    const entities: FilterEntitySchema[] = config.map((entity) => ({
        displayName: entity.displayName,
        attributes: entity.attributes.map(buildAttributeSchema),
    }));

    // Add special CVE filters that aren't part of the regular config
    if (includeSpecialFilters) {
        entities.push({
            displayName: 'CVE Special Filters',
            attributes: [
                {
                    displayName: 'Severity',
                    searchTerm: 'Severity',
                    inputType: 'select',
                    options: [
                        'CRITICAL_VULNERABILITY_SEVERITY',
                        'IMPORTANT_VULNERABILITY_SEVERITY',
                        'MODERATE_VULNERABILITY_SEVERITY',
                        'LOW_VULNERABILITY_SEVERITY',
                    ],
                },
                {
                    displayName: 'Fixable',
                    searchTerm: 'Fixable',
                    inputType: 'select',
                    options: ['true', 'false'],
                },
            ],
        });
    }

    // Generate a description
    const entityNames = entities.map((e) => e.displayName).join(', ');
    const description = `Available filters for: ${entityNames}`;

    return {
        entities,
        description,
    };
}

/**
 * Format filter schema as JSON string for AI prompts
 *
 * @param schema The filter schema
 * @returns JSON string representation
 */
export function formatSchemaForPrompt(schema: FilterSchema): string {
    return JSON.stringify(schema, null, 2);
}

/**
 * Get a list of all available search terms from the schema
 * Useful for validation
 *
 * @param schema The filter schema
 * @returns Array of search terms
 */
export function getAvailableSearchTerms(schema: FilterSchema): string[] {
    const searchTerms: string[] = [];

    schema.entities.forEach((entity) => {
        entity.attributes.forEach((attribute) => {
            searchTerms.push(attribute.searchTerm);
        });
    });

    return searchTerms;
}

/**
 * Get filter schema examples as a formatted string for AI prompts
 * Shows what filters are available and how to use them
 *
 * @param schema The filter schema
 * @returns Formatted examples string
 */
export function getSchemaExamples(schema: FilterSchema): string {
    const examples: string[] = [];

    schema.entities.forEach((entity) => {
        examples.push(`\n${entity.displayName} filters:`);
        entity.attributes.forEach((attribute) => {
            let example = `- ${attribute.displayName} (${attribute.searchTerm})`;

            // Add type-specific information
            if (attribute.inputType === 'date-picker') {
                example += ' - accepts dates';
            } else if (attribute.inputType === 'condition-number') {
                example += ' - accepts numeric conditions (>, <, =, etc.)';
            } else if (attribute.inputType === 'select' && attribute.options) {
                example += ` - options: ${attribute.options.slice(0, 3).join(', ')}${
                    attribute.options.length > 3 ? ', ...' : ''
                }`;
            }

            examples.push(example);
        });
    });

    return examples.join('\n');
}
