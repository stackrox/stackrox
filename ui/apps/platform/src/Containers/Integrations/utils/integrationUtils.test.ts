import { transformDurationLongForm } from './integrationUtils';

describe('integrationUtils', () => {
    describe('transformDurationLongForm', () => {
        it('should return correct result when format is correct', () => {
            let result = transformDurationLongForm('3h');
            expect(result).toBe('3 hours');
            result = transformDurationLongForm('40m');
            expect(result).toBe('40 minutes');
            result = transformDurationLongForm('1s');
            expect(result).toBe('1 second');
            result = transformDurationLongForm('1h1m');
            expect(result).toBe('1 hour 1 minute');
            result = transformDurationLongForm('20m59s');
            expect(result).toBe('20 minutes 59 seconds');
            result = transformDurationLongForm('2h3m4s');
            expect(result).toBe('2 hours 3 minutes 4 seconds');
        });

        it('should return empty when incorrect format', () => {
            let result = transformDurationLongForm('3');
            expect(result).toBe('');
            result = transformDurationLongForm('40f');
            expect(result).toBe('');
            result = transformDurationLongForm('');
            expect(result).toBe('');
        });
    });
});
