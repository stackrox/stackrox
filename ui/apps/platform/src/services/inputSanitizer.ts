/**
 * Input sanitization utilities for natural language search
 * Prevents XSS, injection attacks, and validates input constraints
 */

/**
 * Configuration for input sanitization
 */
export type SanitizationConfig = {
    /** Maximum allowed length in characters */
    maxLength: number;
    /** Whether to strip HTML tags */
    stripHtml: boolean;
    /** Whether to trim whitespace */
    trim: boolean;
};

/**
 * Default sanitization configuration
 */
const DEFAULT_CONFIG: SanitizationConfig = {
    maxLength: 500,
    stripHtml: true,
    trim: true,
};

/**
 * Validation error for sanitization failures
 */
export class InputValidationError extends Error {
    constructor(message: string) {
        super(message);
        this.name = 'InputValidationError';
    }
}

/**
 * Strip HTML tags from input to prevent XSS
 */
function stripHtmlTags(input: string): string {
    // Remove all HTML tags
    return input.replace(/<[^>]*>/g, '');
}

/**
 * Escape special characters that could be used in injection attacks
 */
function escapeSpecialChars(input: string): string {
    // Replace potentially dangerous characters
    return input
        .replace(/[<>'"]/g, (char) => {
            const escapeMap: Record<string, string> = {
                '<': '&lt;',
                '>': '&gt;',
                "'": '&#39;',
                '"': '&quot;',
            };
            return escapeMap[char] || char;
        });
}

/**
 * Validate input length
 */
function validateLength(input: string, maxLength: number): void {
    if (input.length > maxLength) {
        throw new InputValidationError(
            `Input exceeds maximum length of ${maxLength} characters (got ${input.length})`
        );
    }
}

/**
 * Validate that input is not empty after trimming
 */
function validateNotEmpty(input: string): void {
    if (input.length === 0) {
        throw new InputValidationError('Input cannot be empty');
    }
}

/**
 * Sanitize user input for natural language search queries
 *
 * @param input Raw user input
 * @param config Optional sanitization configuration
 * @returns Sanitized input string
 * @throws InputValidationError if input fails validation
 */
export function sanitizeInput(
    input: string,
    config: Partial<SanitizationConfig> = {}
): string {
    const finalConfig = { ...DEFAULT_CONFIG, ...config };

    // Start with original input
    let sanitized = input;

    // Step 1: Trim whitespace if configured
    if (finalConfig.trim) {
        sanitized = sanitized.trim();
    }

    // Step 2: Validate not empty
    validateNotEmpty(sanitized);

    // Step 3: Validate length
    validateLength(sanitized, finalConfig.maxLength);

    // Step 4: Strip HTML tags if configured
    if (finalConfig.stripHtml) {
        sanitized = stripHtmlTags(sanitized);
    }

    // Step 5: Escape special characters
    sanitized = escapeSpecialChars(sanitized);

    // Step 6: Final validation after sanitization
    validateNotEmpty(sanitized);

    return sanitized;
}

/**
 * Check if input is valid without throwing errors
 *
 * @param input Raw user input
 * @param config Optional sanitization configuration
 * @returns True if input is valid, false otherwise
 */
export function isValidInput(
    input: string,
    config: Partial<SanitizationConfig> = {}
): boolean {
    try {
        sanitizeInput(input, config);
        return true;
    } catch {
        return false;
    }
}
