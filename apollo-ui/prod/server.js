var express = require('express');
var proxy = require('http-proxy-middleware');
var app = express();

var endpoint = process.env.ROX_APOLLO_ENDPOINT || "apollo.apollo_net:8080";

app.use(express.static('./build'));
app.use('/v1', proxy({
    target: "https://" + endpoint,
    changeOrigin: true,
    secure: false
}));
app.listen(3000);