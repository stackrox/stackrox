FROM --platform=linux/amd64 nginx

# Lazy build, just copy assets from local dir to avoid mono-repo pain
RUN chmod g+rwx /var/cache/nginx /var/run /var/log/nginx
COPY ./dist /usr/share/nginx/html
