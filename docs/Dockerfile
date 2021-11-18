FROM registry.access.redhat.com/ubi8/nginx-118:1-46

# Delete unnecessary things from the content directory.
RUN find . -mindepth 1 -delete

COPY ./build/ ./

EXPOSE 8080

CMD nginx -g "daemon off;"
