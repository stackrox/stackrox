// Use this script to generate a new fragmentTypes.json. In a new folder:
// 1. Put this file in a new local folder
// 2. Replace 'YOUR_TOKEN_HERE' with your localhost auth token
// 3. yarn init .
// 4. yarn add node-fetch
// 5. node getGQLSchema.js
// 6. Copy newly created fragmentTypes.json to /rox/ui/src

const fetch = require('node-fetch');
const fs = require('fs');

const authorization = 'YOUR_TOKEN_HERE';

process.env.NODE_TLS_REJECT_UNAUTHORIZED = 0;
fetch(`https://localhost:3000/api/graphql`, {
    method: 'POST',
    headers: {
        'Content-Type': 'application/json',
        Authorization: authorization
    },

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
    `
    })
})
    .then(result => result.json())
    .then(result => {
        const filteredData = result.data.__schema.types.filter(type => type.possibleTypes !== null);
        result.data.__schema.types = filteredData;

        fs.writeFile('./fragmentTypes.json', JSON.stringify(result.data), err => {
            if (err) {
                console.error('Error writing fragmentTypes file', err);
            } else {
                console.log('Fragment types successfully extracted!');
            }
        });
    });
