/**
 * Mostly copied from https://www.apollographql.com/docs/react/data/fragments/#generating-possibletypes-automatically
 *
 * This script is used to produce `src/possibleTypes.json`.
 * It's preferred to run the corresponding `package.json` script instead of invoking this script directly.
 */

const fetch = require('node-fetch');
const fs = require('fs');

const authToken = process.env.ROX_AUTH_TOKEN;
const apiEndpoint = process.env.UI_BASE_URL || 'https://localhost:8000';
const graphqlApiEndpoint = `${apiEndpoint}/api/graphql`;
const outputFile = process.env.GRAPHQL_POSSIBLE_TYPES_FILE || './src/possibleTypes.json';

const authTokenMasked = authToken ? `${authToken.slice(0, 3)}..${authToken.slice(-3)}` : 'N/A';
console.log(
    `Retrieving GraphQL fragment types from '${graphqlApiEndpoint}', auth token: ${authTokenMasked}.`
);
console.log(`The output will be saved to '${outputFile}' file.`);

process.env.NODE_TLS_REJECT_UNAUTHORIZED = 0;
fetch(graphqlApiEndpoint, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${authToken}` },
    body: JSON.stringify({
        variables: {},
        query: `
      {
        __schema {
          types {
            kind
            name
            possibleTypes {
              name
            }
          }
        }
      }
    `,
    }),
})
    .then((result) => result.json())
    .then((result) => {
        const possibleTypes = {};

        result.data.__schema.types.forEach((supertype) => {
            if (supertype.possibleTypes) {
                possibleTypes[supertype.name] = supertype.possibleTypes.map(
                    (subtype) => subtype.name
                );
            }
        });

        fs.writeFile(outputFile, JSON.stringify(possibleTypes), (err) => {
            if (err) {
                console.error(`Error writing '${outputFile}'`, err);
            } else {
                console.log('Fragment types successfully extracted and saved!');
            }
        });
    });
