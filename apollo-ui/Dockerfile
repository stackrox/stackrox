FROM mhart/alpine-node

# Create app directory
WORKDIR /usr/src/app

# Install app dependencies
# A wildcard is used to ensure both package.json AND package-lock.json are copied
# where available (npm@5+)
COPY build ./

# If you are building your code for production
# RUN npm install --only=production
RUN npm install -g serve
RUN yarn install

EXPOSE 3000
CMD [ "serve", "-p 3000 -s build" ]
