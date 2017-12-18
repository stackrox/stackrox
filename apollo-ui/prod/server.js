const express = require('express');
const proxy = require('http-proxy-middleware');

const app = express();

const endpoint = process.env.ROX_APOLLO_ENDPOINT || 'apollo.apollo_net:8080';

app.use(express.static(`${__dirname}/build`));

app.use('/v1', proxy({
    target: `https://${endpoint}`,
    changeOrigin: true,
    secure: false
}));

// redirect all dynamically added URLs to index.html (let react app to handle 404.)
app.get('/*', (req, res) => {
    res.sendFile(`${__dirname}/build/index.html`);
});

app.listen(3000);
