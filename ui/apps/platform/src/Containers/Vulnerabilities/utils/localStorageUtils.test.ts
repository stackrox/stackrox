import { isVulnMgmtLocalStorage } from '../types';
import { normalizeLocalStorageKeys } from './localStorageUtils';

describe('normalizeLocalStorageKeys', () => {
    it('should rename legacy SEVERITY key to Severity', () => {
        const input = {
            preferences: {
                defaultFilters: {
                    SEVERITY: ['Critical', 'Important'],
                    Fixable: ['Fixable'],
                },
            },
        };
        expect(normalizeLocalStorageKeys(input)).toEqual({
            preferences: {
                defaultFilters: {
                    Severity: ['Critical', 'Important'],
                    Fixable: ['Fixable'],
                },
            },
        });
    });

    it('should rename legacy FIXABLE key to Fixable', () => {
        const input = {
            preferences: {
                defaultFilters: {
                    Severity: ['Critical'],
                    FIXABLE: ['Fixable'],
                },
            },
        };
        expect(normalizeLocalStorageKeys(input)).toEqual({
            preferences: {
                defaultFilters: {
                    Severity: ['Critical'],
                    Fixable: ['Fixable'],
                },
            },
        });
    });

    it('should rename both SEVERITY and FIXABLE together', () => {
        const input = {
            preferences: {
                defaultFilters: {
                    SEVERITY: ['Critical', 'Important'],
                    FIXABLE: ['Fixable'],
                },
            },
        };
        expect(normalizeLocalStorageKeys(input)).toEqual({
            preferences: {
                defaultFilters: {
                    Severity: ['Critical', 'Important'],
                    Fixable: ['Fixable'],
                },
            },
        });
    });

    it('should not modify data that already uses canonical keys', () => {
        const input = {
            preferences: {
                defaultFilters: {
                    Severity: ['Critical', 'Important'],
                    Fixable: ['Fixable'],
                },
            },
        };
        expect(normalizeLocalStorageKeys(input)).toBe(input);
    });

    it('should prefer existing key with correct casing when both casings exist', () => {
        const input = {
            preferences: {
                defaultFilters: {
                    SEVERITY: ['Low'],
                    Severity: ['Critical'],
                    Fixable: ['Fixable'],
                },
            },
        };
        const result = normalizeLocalStorageKeys(input);
        expect(result).toEqual({
            preferences: {
                defaultFilters: {
                    Severity: ['Critical'],
                    Fixable: ['Fixable'],
                },
            },
        });
    });

    it('should return non-object values unchanged', () => {
        expect(normalizeLocalStorageKeys(null)).toBeNull();
        expect(normalizeLocalStorageKeys('string')).toBe('string');
        expect(normalizeLocalStorageKeys(42)).toBe(42);
    });

    it('should return objects with missing structure unchanged', () => {
        expect(normalizeLocalStorageKeys({})).toEqual({});
        expect(normalizeLocalStorageKeys({ preferences: 'invalid' })).toEqual({
            preferences: 'invalid',
        });
        expect(normalizeLocalStorageKeys({ preferences: {} })).toEqual({ preferences: {} });
    });
});

describe('normalizeLocalStorageKeys + isVulnMgmtLocalStorage', () => {
    it('should validate data with legacy keys after normalization', () => {
        const legacyData = {
            preferences: {
                defaultFilters: {
                    SEVERITY: ['Critical', 'Important'],
                    FIXABLE: ['Fixable'],
                },
            },
        };
        const normalized = normalizeLocalStorageKeys(legacyData);
        expect(isVulnMgmtLocalStorage(normalized)).toBe(true);
    });

    it('should validate data with canonical keys directly', () => {
        const data = {
            preferences: {
                defaultFilters: {
                    Severity: ['Critical', 'Important'],
                    Fixable: ['Fixable'],
                },
            },
        };
        expect(isVulnMgmtLocalStorage(data)).toBe(true);
    });

    it('should reject data with legacy keys without normalization', () => {
        const legacyData = {
            preferences: {
                defaultFilters: {
                    SEVERITY: ['Critical', 'Important'],
                    FIXABLE: ['Fixable'],
                },
            },
        };
        expect(isVulnMgmtLocalStorage(legacyData)).toBe(false);
    });
});
