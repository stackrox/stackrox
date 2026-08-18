import { defaultValues, validationSchema } from './LightspeedIntegrationForm';

describe('LightspeedIntegrationForm validationSchema', () => {
    it('passes with a name and a service URL', () => {
        const value = {
            integration: {
                id: '',
                name: 'OpenShift Lightspeed',
                type: 'AI_INTEGRATION_TYPE_OLS',
                serviceUrl: 'https://lightspeed.example.com',
            },
        };
        expect(validationSchema.isValidSync(value)).toBe(true);
    });

    it('passes when the service URL is empty because it is optional', () => {
        const value = {
            integration: {
                id: '',
                name: 'OpenShift Lightspeed',
                type: 'AI_INTEGRATION_TYPE_OLS',
                serviceUrl: '',
            },
        };
        expect(validationSchema.isValidSync(value)).toBe(true);
    });

    it('fails when the name is missing', () => {
        const value = {
            integration: {
                id: '',
                name: '',
                type: 'AI_INTEGRATION_TYPE_OLS',
                serviceUrl: 'https://lightspeed.example.com',
            },
        };
        expect(validationSchema.isValidSync(value)).toBe(false);
    });

    it('fails when the name is only whitespace', () => {
        const value = {
            integration: {
                id: '',
                name: '   ',
                type: 'AI_INTEGRATION_TYPE_OLS',
                serviceUrl: '',
            },
        };
        expect(validationSchema.isValidSync(value)).toBe(false);
    });

    it('fails when the type is not the OLS type', () => {
        const value = {
            integration: {
                id: '',
                name: 'OpenShift Lightspeed',
                type: 'AI_INTEGRATION_TYPE_UNSPECIFIED',
                serviceUrl: '',
            },
        };
        expect(validationSchema.isValidSync(value)).toBe(false);
    });

    it('rejects the empty default values because a name is required', () => {
        expect(validationSchema.isValidSync(defaultValues)).toBe(false);
    });
});
