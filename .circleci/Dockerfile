FROM circleci/golang:1.9.2

# Install Bazel
RUN echo "deb [arch=amd64] http://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee /etc/apt/sources.list.d/bazel.list && \
    curl https://bazel.build/bazel-release.pub.gpg | sudo apt-key add - && \
    sudo apt-get update && \
    sudo apt-get install -y \
        bazel \
        openjdk-8-jdk \
        && \
    sudo apt-get autoremove -y

# Add necessary Go build tools
RUN go get -u github.com/golang/lint/golint && \
    go get -u golang.org/x/tools/cmd/goimports && \
    go get -u github.com/jstemmer/go-junit-report && \
    curl -L -o $GOPATH/bin/dep https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64 && \
    chmod +x $GOPATH/bin/dep
