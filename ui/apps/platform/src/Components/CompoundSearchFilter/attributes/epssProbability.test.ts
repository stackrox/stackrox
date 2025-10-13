import {
    EPSSProbability,
    convertFromExternalToInternalText,
    convertFromInternalToExternalText,
    externalTextDefault,
    externalTextRegExp,
    internalTextRegExp,
    validateExternalText,
    validateInternalText,
} from './epssProbability';
import { convertFromInternalToExternalConditionText } from '../components/ConditionText';

describe('epssProbability', () => {
    describe('convertFromExternalToInternalText', () => {
        it('should convert one digit without period', () => {
            expect(convertFromExternalToInternalText('0')).toEqual('0');
        });

        it('should convert two digits with period', () => {
            // Make asserttion independent of exact value 0.12300000000000001
            expect(convertFromExternalToInternalText('12.3').startsWith('0.123')).toEqual(true);
        });

        it('should convert three digits without period', () => {
            expect(convertFromExternalToInternalText('100')).toEqual('1');
        });

        it('should convert period preceded by digit but without percent', () => {
            expect(convertFromExternalToInternalText('0.')).toEqual('0');
        });

        it('should convert period followed by digit but without percent', () => {
            expect(convertFromExternalToInternalText('.0')).toEqual('0');
        });

        it('should convert period followed by digits but without percent', () => {
            expect(convertFromExternalToInternalText('.123')).toEqual('0.00123');
        });

        it('should convert period preceded and followed by digit but without percent', () => {
            expect(convertFromExternalToInternalText('0.0')).toEqual('0');
        });

        it('should convert period preceded by digit and with percent', () => {
            expect(convertFromExternalToInternalText('0.%')).toEqual('0');
        });

        it('should convert period followed by digits and with percent', () => {
            expect(convertFromExternalToInternalText('.123%')).toEqual('0.00123');
        });

        it('should convert period followed by digit and with percent', () => {
            expect(convertFromExternalToInternalText('.0%')).toEqual('0');
        });

        it('should convert period preceded and followed by digit and with percent', () => {
            expect(convertFromExternalToInternalText('0.0%')).toEqual('0');
        });

        it('should convert externalTextDefault', () => {
            expect(convertFromExternalToInternalText(externalTextDefault)).toEqual('0');
        });

        it('should convert valid digits but with preceding space', () => {
            expect(convertFromExternalToInternalText(' 0.')).toEqual('0');
        });

        it('should convert valid digits but with following space', () => {
            expect(convertFromExternalToInternalText('.0 ')).toEqual('0');
        });

        it('should convert valid digits but with preceding and following space', () => {
            expect(convertFromExternalToInternalText(' 0.0 ')).toEqual('0');
        });
    });

    describe('convertFromInternalToExternalText', () => {
        it('should convert zero without period', () => {
            expect(convertFromInternalToExternalText('0')).toEqual('0.000%');
        });

        it('should convert one without period', () => {
            expect(convertFromInternalToExternalText('1')).toEqual('100.000%');
        });

        it('should convert period preceded by digit', () => {
            expect(convertFromInternalToExternalText('0.')).toEqual('0.000%');
        });

        it('should convert period followed by digit', () => {
            expect(convertFromInternalToExternalText('.0')).toEqual('0.000%');
        });

        it('should convert period followed by three digits', () => {
            expect(convertFromInternalToExternalText('.123')).toEqual('12.300%');
        });

        it('should convert period followed by five digits', () => {
            expect(convertFromInternalToExternalText('.12345')).toEqual('12.345%');
        });

        it('should convert period followed by six digits', () => {
            // toFixed(3) rounds up, if needed
            expect(convertFromInternalToExternalText('.123456')).toEqual('12.346%');
        });

        it('should convert period preceded and followed by digit but without percent', () => {
            expect(convertFromInternalToExternalText('0.0')).toEqual('0.000%');
        });

        it('should convert valid digits but with preceding space', () => {
            expect(convertFromInternalToExternalText(' 0.')).toEqual('0.000%');
        });

        it('should convert valid digits but with following space', () => {
            expect(convertFromExternalToInternalText('.0 ')).toEqual('0');
        });

        it('should convert valid digits but with preceding and following space', () => {
            expect(convertFromInternalToExternalText(' 0.0 ')).toEqual('0.000%');
        });
    });

    describe('externalTextRegExp negative', () => {
        it('should fail for empty string', () => {
            expect(externalTextRegExp.test('')).toEqual(false);
        });

        it('should fail for bogus text', () => {
            expect(externalTextRegExp.test('bogus')).toEqual(false);
        });

        it('should fail for period alone', () => {
            expect(externalTextRegExp.test('.')).toEqual(false);
        });

        it('should fail for percent alone', () => {
            expect(externalTextRegExp.test('%')).toEqual(false);
        });

        it('should fail for negative number', () => {
            expect(externalTextRegExp.test('-1')).toEqual(false);
        });

        // Why fail because of space?
        // To reduce complexity of RegExp, convert function is responsible to trim.

        it('should fail for valid digits but with preceding space', () => {
            expect(externalTextRegExp.test(' 0.')).toEqual(false);
        });

        it('should fail for valid digits but with following space', () => {
            expect(externalTextRegExp.test('.0 ')).toEqual(false);
        });

        it('should fail for valid digits but with preceding and following space', () => {
            expect(externalTextRegExp.test(' 0.0 ')).toEqual(false);
        });
    });

    describe('externalTextRegExp positive', () => {
        it('should pass for one digit without period', () => {
            expect(externalTextRegExp.test('0')).toEqual(true);
        });

        it('should pass for two digits with period', () => {
            expect(externalTextRegExp.test('12.3')).toEqual(true);
        });

        it('should pass for three digits without period', () => {
            expect(externalTextRegExp.test('100')).toEqual(true);
        });

        it('should pass for four digits with period', () => {
            // To reduce complexity of RegExp, convert function is responsible for range.
            expect(externalTextRegExp.test('1234.567')).toEqual(true);
        });

        it('should pass for period preceded by digit but without percent', () => {
            expect(externalTextRegExp.test('0.')).toEqual(true);
        });

        it('should pass for period followed by digit but without percent', () => {
            expect(externalTextRegExp.test('.0')).toEqual(true);
        });

        it('should pass for period followed by digits but without percent', () => {
            expect(externalTextRegExp.test('.123')).toEqual(true);
        });

        it('should pass for period preceded and followed by digit but without percent', () => {
            expect(externalTextRegExp.test('0.0')).toEqual(true);
        });

        it('should pass for period preceded by digit and with percent', () => {
            expect(externalTextRegExp.test('0.%')).toEqual(true);
        });
        it('should pass for period followed by digits and with percent', () => {
            expect(externalTextRegExp.test('.123%')).toEqual(true);
        });

        it('should pass for period followed by digit and with percent', () => {
            expect(externalTextRegExp.test('.0%')).toEqual(true);
        });

        it('should pass for period preceded and followed by digit and with percent', () => {
            expect(externalTextRegExp.test('0.0%')).toEqual(true);
        });

        it('should pass for externalTextDefault', () => {
            expect(externalTextRegExp.test(externalTextDefault)).toEqual(true);
        });
    });

    describe('internalTextRegExp negative', () => {
        it('should fail for empty string', () => {
            expect(internalTextRegExp.test('')).toEqual(false);
        });

        it('should fail for bogus text', () => {
            expect(internalTextRegExp.test('bogus')).toEqual(false);
        });

        it('should fail for period alone', () => {
            expect(internalTextRegExp.test('.')).toEqual(false);
        });

        it('should fail for percent alone', () => {
            expect(internalTextRegExp.test('%')).toEqual(false);
        });

        it('should fail for negative number', () => {
            expect(internalTextRegExp.test('-1')).toEqual(false);
        });

        it('should fail for period preceded by digit and with percent', () => {
            expect(internalTextRegExp.test('0.%')).toEqual(false);
        });

        it('should fail for period followed by digit and with percent', () => {
            expect(internalTextRegExp.test('.0%')).toEqual(false);
        });

        it('should fail for period preceded and followed by digit and with percent', () => {
            expect(internalTextRegExp.test('0.0%')).toEqual(false);
        });

        // Why fail because of space?
        // To reduce complexity of RegExp, convert function is responsible to trim.

        it('should fail for valid digits but with preceding space', () => {
            expect(internalTextRegExp.test(' 0.')).toEqual(false);
        });

        it('should fail for valid digits but with following space', () => {
            expect(internalTextRegExp.test('.0 ')).toEqual(false);
        });

        it('should fail for valid digits but with preceding and following space', () => {
            expect(internalTextRegExp.test(' 0.0 ')).toEqual(false);
        });
    });

    describe('internalTextRegExp positive', () => {
        it('should pass for one digit without period', () => {
            expect(internalTextRegExp.test('0')).toEqual(true);
        });

        it('should pass for two digits with period', () => {
            // To reduce complexity of RegExp, convert function is responsible for range.
            expect(internalTextRegExp.test('12.3')).toEqual(true);
        });

        it('should pass for period preceded by digit but without percent', () => {
            expect(internalTextRegExp.test('0.')).toEqual(true);
        });

        it('should pass for period followed by digit but without percent', () => {
            expect(internalTextRegExp.test('.0')).toEqual(true);
        });

        it('should pass for period followed by digits but without percent', () => {
            expect(internalTextRegExp.test('.123')).toEqual(true);
        });

        it('should pass for period preceded and followed by digit but without percent', () => {
            expect(internalTextRegExp.test('0.0')).toEqual(true);
        });
    });

    describe('validateExternalText negative', () => {
        it('should fail for number that is too large but without percent', () => {
            expect(validateExternalText('101')).toEqual(false);
        });

        it('should fail for number that is too large and with percent', () => {
            expect(validateExternalText('101$')).toEqual(false);
        });
    });

    describe('validateExternalText positive', () => {
        it('should pass for valid digits but with preceding space', () => {
            expect(validateExternalText(' 0.')).toEqual(true);
        });

        it('should pass for valid digits but with following space', () => {
            expect(validateExternalText('.0 ')).toEqual(true);
        });

        it('should pass for valid digits but with preceding and following space', () => {
            expect(validateExternalText(' 0.0 ')).toEqual(true);
        });

        it('should pass for externalTextDefault', () => {
            expect(validateExternalText(externalTextDefault)).toEqual(true);
        });
    });

    describe('validateInternalText negative', () => {
        it('should fail for number that is too large but without percent', () => {
            expect(validateInternalText('1.01')).toEqual(false);
        });

        it('should fail for number that is too large and with percent', () => {
            expect(validateInternalText('1.01$')).toEqual(false);
        });
    });

    describe('validateInternalText positive', () => {
        it('should pass for valid digits but with preceding space', () => {
            expect(validateInternalText(' 0.')).toEqual(true);
        });

        it('should pass for valid digits but with following space', () => {
            expect(validateInternalText('.0 ')).toEqual(true);
        });

        it('should pass for valid digits but with preceding and following space', () => {
            expect(validateInternalText(' 0.0 ')).toEqual(true);
        });
    });
});

// For filter chip: split, convert, and then join.
// EPSSProbability has specific functions to test genmric function in component.

describe('convertFromInternalToExternalConditionText negative', () => {
    const { inputProps } = EPSSProbability;

    // Notice typograpical quotes.

    it('should render empty string as not valid', () => {
        expect(convertFromInternalToExternalConditionText(inputProps, '')).toEqual(
            '“” is not valid'
        );
    });

    it('should render bogus text as not valid', () => {
        expect(convertFromInternalToExternalConditionText(inputProps, 'bogus')).toEqual(
            '“bogus” is not valid'
        );
    });

    it('should render absence of condition as not valid', () => {
        expect(convertFromInternalToExternalConditionText(inputProps, '0')).toEqual(
            '“0” is not valid'
        );
    });

    it('should render unexpected condition as not valid', () => {
        expect(convertFromInternalToExternalConditionText(inputProps, '~0')).toEqual(
            '“~0” is not valid'
        );
    });

    it('should render equal to as not valid', () => {
        // Intentionally omit = because potential problem with floating point
        expect(convertFromInternalToExternalConditionText(inputProps, '=0.5')).toEqual(
            '“=0.5” is not valid'
        );
    });

    it('should render unexpected value as not valid', () => {
        expect(convertFromInternalToExternalConditionText(inputProps, '=1.1')).toEqual(
            '“=1.1” is not valid'
        );
    });
});

describe('convertFromInternalToExternalConditionText positive', () => {
    const { inputProps } = EPSSProbability;

    it('should split greater than', () => {
        expect(convertFromInternalToExternalConditionText(inputProps, '>0')).toEqual('>0.000%');
    });

    it('should split greater than or equal to', () => {
        // toFixed(3) did not need to round up apparently (however, see next test)
        expect(convertFromInternalToExternalConditionText(inputProps, '>=.012345')).toEqual(
            '>=1.234%'
        );
    });

    it('should round up if needed', () => {
        // toFixed(3) rounds up, if needed
        expect(convertFromInternalToExternalConditionText(inputProps, '>=.023456')).toEqual(
            '>=2.346%'
        );
    });

    // Intentionally omit = because potential problem with floating point

    it('should split less than or equal to', () => {
        expect(convertFromInternalToExternalConditionText(inputProps, '<=0.00345')).toEqual(
            '<=0.345%'
        );
    });

    it('should split less than', () => {
        expect(convertFromInternalToExternalConditionText(inputProps, '<1')).toEqual('<100.000%');
    });
});
