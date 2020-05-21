import { setStoredCredentialsField, setFormSubmissionOptions } from './integrationFormUtils';

describe('integrationFormUtils', () => {
    describe('setStoredCredentialsField', () => {
        it('should add the "hasStoredCredentials" field for an integration', () => {
            const source = 'imageIntegrations';
            const type = 'docker';
            const initialValues = {
                docker: {
                    password: '******',
                },
            };

            expect(setStoredCredentialsField(source, type, initialValues)).toEqual({
                hasStoredCredentials: true,
                docker: {
                    password: '',
                },
            });
        });

        it('should not add the "hasStoredCredentials" field for an integration', () => {
            const source = 'notifiers';
            const type = 'slack';
            const initialValues = {
                docker: {
                    password: '',
                },
            };

            expect(setStoredCredentialsField(source, type, initialValues)).toEqual(initialValues);
        });
    });

    describe('setFormSubmissionOptions', () => {
        it('should return options with the updatePassword set to true', () => {
            const source = 'imageIntegrations';
            const type = 'docker';
            const data = {
                docker: {
                    password: 'NEW_PASSWORD',
                },
            };

            expect(setFormSubmissionOptions(source, type, data)).toEqual({
                updatePassword: true,
            });
            expect(
                setFormSubmissionOptions(source, type, data, { isNewIntegration: false })
            ).toEqual({
                updatePassword: true,
            });
        });

        it('should return options with the updatePassword set to false', () => {
            const source = 'imageIntegrations';
            const type = 'docker';
            const data = {
                docker: {
                    password: '',
                },
            };

            expect(setFormSubmissionOptions(source, type, data)).toEqual({
                updatePassword: false,
            });
            expect(
                setFormSubmissionOptions(source, type, data, { isNewIntegration: false })
            ).toEqual({
                updatePassword: false,
            });
        });

        it('should return options with the updatePassword set to true for a new integration', () => {
            const source = 'imageIntegrations';
            const type = 'docker';
            const data = {
                docker: {
                    password: '',
                },
            };

            expect(
                setFormSubmissionOptions(source, type, data, { isNewIntegration: true })
            ).toEqual({
                updatePassword: true,
            });
        });

        it('should not return options with the updatePassword set', () => {
            const source = 'notifiers';
            const type = 'slack';
            const data = {};

            expect(setFormSubmissionOptions(source, type, data)).toEqual(null);
        });
    });
});
