import { setStoredCredentialFields, setFormSubmissionOptions } from './integrationFormUtils';

describe('integrationFormUtils', () => {
    describe('setStoredCredentialFields', () => {
        it('should add the "hasStoredCredentials" field and clear the field for an integration', () => {
            const source = 'imageIntegrations';
            const type = 'ecr';
            const initialValues = {
                ecr: {
                    accessKeyId: '******',
                    secretAccessKey: '******',
                },
            };

            expect(setStoredCredentialFields(source, type, initialValues)).toEqual({
                hasStoredCredentials: true,
                ecr: {
                    accessKeyId: '',
                    secretAccessKey: '',
                },
            });
        });

        it('should not add the "hasStoredCredentials" field for an integration', () => {
            const source = 'imageIntegrations';
            const type = 'ecr';
            const initialValues = {
                ecr: {
                    accessKeyId: '',
                    secretAccessKey: '',
                },
            };

            expect(setStoredCredentialFields(source, type, initialValues)).toEqual(initialValues);
        });
    });

    describe('setFormSubmissionOptions', () => {
        it('should return options with the updatePassword set to true', () => {
            const source = 'imageIntegrations';
            const type = 'ecr';
            const data = {
                ecr: {
                    accessKeyId: 'NEW_CREDENTIALS',
                    secretAccessKey: 'NEW_CREDENTIALS',
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
            const type = 'ecr';
            const data = {
                ecr: {
                    accessKeyId: '',
                    secretAccessKey: '',
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
            const type = 'ecr';
            const data = {
                ecr: {
                    accessKeyId: '',
                    secretAccessKey: '',
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
