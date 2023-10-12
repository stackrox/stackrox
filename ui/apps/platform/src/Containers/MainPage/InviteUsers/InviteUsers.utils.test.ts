import { AuthProvider } from 'services/AuthService';

import { splitEmailsIntoNewAndExisting } from './InviteUsers.utils';

const authProviderWithGroups: AuthProvider = {
    id: '396cece6-16c2-486b-a659-c383aeb52216',
    name: 'MysteryInc',
    type: 'auth0',
    uiEndpoint: 'localhost:3000',
    enabled: true,
    config: {
        client_id: 'rQgF3pSeKc1nCAWESLd17af8BtsE7cZO',
        client_secret: '',
        issuer: 'https://sr-dev.auth0.com',
        mode: 'fragment',
    },
    loginUrl: '/sso/login/396cece6-16c2-486b-a659-c383aeb52216',
    extraUiEndpoints: [],
    active: false,
    requiredAttributes: [],
    traits: {
        mutabilityMode: 'ALLOW_MUTATE',
        visibility: 'VISIBLE',
        origin: 'IMPERATIVE',
    },
    claimMappings: {},
    lastUpdated: '2023-09-27T20:01:54.243147762Z',
    groups: [
        {
            props: {
                id: 'io.stackrox.authz.group.d113debb-d527-4a84-bcfb-f581cc4a6e15',
                traits: null,
                authProviderId: '396cece6-16c2-486b-a659-c383aeb52216',
                key: 'email',
                value: 'scooby@redhat.com',
            },
            roleName: 'Admin',
        },
        {
            props: {
                id: 'io.stackrox.authz.group.d113debb-d527-4a84-bcfb-f581cc4a6e16',
                traits: null,
                authProviderId: '396cece6-16c2-486b-a659-c383aeb52216',
                key: 'email',
                value: 'shaggy@redhat.com',
            },
            roleName: 'Admin',
        },
    ],
    defaultRole: 'Analyst',
};

describe('splitEmailsIntoNewAndExisting', () => {
    it('should empty arrays when no emails are passed in', () => {
        const emails = [];

        const buckets = splitEmailsIntoNewAndExisting(authProviderWithGroups, emails);

        expect(buckets).toEqual({ newEmails: [], existingEmails: [] });
    });

    it('should split emails that already appear in groups from those that do not', () => {
        const emails = ['velma@redhat.com', 'scooby@redhat.com', 'freedy@redhat.com'];

        const buckets = splitEmailsIntoNewAndExisting(authProviderWithGroups, emails);

        expect(buckets).toEqual({
            newEmails: ['velma@redhat.com', 'freedy@redhat.com'],
            existingEmails: ['scooby@redhat.com'],
        });
    });
});
