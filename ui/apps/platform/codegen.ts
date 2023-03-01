const token = process.env.ROX_AUTH_TOKEN;
const baseUrl = process.env.UI_BASE_URL ?? 'https://localhost:3000';
const gqlUrl = `${baseUrl}/api/graphql`;

if (!token) {
    throw new Error(
        'A valid auth token must be defined in the `ROX_AUTH_TOKEN` environment variable'
    );
}

// TODO Problems?
//
// - Different GQL responses depending on feature flags
// - Bundle sizes, no code splitting (see if we can get babel plugin working)

const config = {
    schema: [
        {
            [gqlUrl]: {
                headers: {
                    Authorization: `Bearer ${token}`,
                },
            },
        },
    ],
    documents: ['src/**/*.tsx', 'src/**/*.ts'],
    generates: {
        './src/gql/': {
            preset: 'client',
        },
    },
    hooks: { afterAllFileWrite: ['prettier --write'] },
};

export default config;
