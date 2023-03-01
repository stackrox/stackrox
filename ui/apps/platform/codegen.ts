// eslint-disable-next-line import/no-extraneous-dependencies
import { CodegenConfig } from '@graphql-codegen/cli';

const token = process.env.ROX_AUTH_TOKEN;
const baseUrl = process.env.UI_BASE_URL ?? 'https://localhost:3000';
const gqlUrl = `${baseUrl}/api/graphql`;

if (!token) {
    throw new Error(
        'A valid auth token must be defined in the `ROX_AUTH_TOKEN` environment variable'
    );
}

const config: CodegenConfig = {
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
        './src/generated/graphql-codegen/': {
            preset: 'client',
            presetConfig: {
                fragmentMasking: false,
            },
            config: {
                scalars: {
                    // Our backend sends a `Time` type that is a string in RFC 3339 format
                    Time: 'string',
                },
            },
        },
    },
    hooks: { afterAllFileWrite: ['prettier --write'] },
};

export default config;
