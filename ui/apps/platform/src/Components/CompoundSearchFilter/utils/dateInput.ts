import { format } from 'date-fns';

/**
 * Formats a date for display in the date-picker inputs.
 * @param date - The date to format
 * @returns Date string like "01/15/2034"
 */
export function dateFormat(date: Date): string {
    return format(date, 'MM/DD/YYYY');
}

/**
 * Parses a strict MM/DD/YYYY date string into a local-time Date.
 * @param date - Date string to parse
 * @returns The parsed date, or an invalid Date if the string does not match the format
 */
export function dateParse(date: string): Date {
    const split = date.split('/');
    if (split.length !== 3) {
        return new Date('Invalid Date');
    }
    const [month, day, year] = split;
    if (month.length !== 2 || day.length !== 2 || year.length !== 4) {
        return new Date('Invalid Date');
    }
    return new Date(`${year}-${month}-${day}T00:00:00`);
}

/**
 * Returns whether a Date holds a real date value.
 * @param date - The date to check
 * @returns True if the date is valid
 */
export function isValidDate(date: Date): boolean {
    return !Number.isNaN(date.getTime());
}
