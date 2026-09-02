import { defaultValues, validationSchema } from './LightspeedIntegrationForm';
import type { AiIntegrationFormValues } from './LightspeedIntegrationForm';

function makeIntegration(
    overrides: Partial<AiIntegrationFormValues> = {}
): AiIntegrationFormValues {
    return {
        id: '',
        name: 'OpenShift Lightspeed',
        type: 'AI_INTEGRATION_TYPE_OLS',
        serviceUrl: 'https://lightspeed.example.com',
        ...overrides,
    };
}

describe('LightspeedIntegrationForm validationSchema', () => {
    it('passes with a name and a service URL', () => {
        expect(validationSchema.isValidSync(makeIntegration())).toBe(true);
    });

    it('passes when the service URL is empty because it is optional', () => {
        expect(validationSchema.isValidSync(makeIntegration({ serviceUrl: '' }))).toBe(true);
    });

    it('fails when the name is missing', () => {
        expect(validationSchema.isValidSync(makeIntegration({ name: '' }))).toBe(false);
    });

    it('fails when the name is only whitespace', () => {
        expect(validationSchema.isValidSync(makeIntegration({ name: '   ' }))).toBe(false);
    });

    it('fails when the type is not the OLS type', () => {
        expect(
            validationSchema.isValidSync(
                makeIntegration({ type: 'AI_INTEGRATION_TYPE_UNSPECIFIED' })
            )
        ).toBe(false);
    });

    it('rejects the empty default values because a name is required', () => {
        expect(validationSchema.isValidSync(defaultValues)).toBe(false);
    });
});
