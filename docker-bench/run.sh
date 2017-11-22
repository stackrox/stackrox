 docker run -e ROX_APOLLO_ENDPOINT=localhost:8080 -it --net host --pid host --cap-add audit_control     -e DOCKER_CONTENT_TRUST=$DOCKER_CONTENT_TRUST  -v /var/lib:/var/lib     -v /var/run/docker.sock:/var/run/docker.sock     -v /usr/lib/systemd:/usr/lib/systemd -v /lib/systemd:/lib/systemd  -v /etc:/etc -v /var/log/audit:/var/log/audit   stackrox/docker-bench:latest

