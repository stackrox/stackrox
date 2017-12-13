FROM mhart/alpine-node

# Create app directory
WORKDIR /usr/src/app

# Install app dependencies
# A wildcard is used to ensure both package.json AND package-lock.json are copied
# where available (npm@5+)
COPY build ./build
COPY prod/package.json ./
COPY prod/server.js ./
COPY prod/yarn.lock ./

RUN yarn install

EXPOSE 3000
CMD [ "npm", "run", "serve" ]
