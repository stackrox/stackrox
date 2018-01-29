FROM circleci/golang:1.9.2

# Install Bazel
RUN echo "deb [arch=amd64] http://storage.googleapis.com/bazel-apt stable jdk1.8" | sudo tee /etc/apt/sources.list.d/bazel.list && \
    curl https://bazel.build/bazel-release.pub.gpg | sudo apt-key add - && \
    sudo apt-get update && \
    sudo apt-get install -y \
        bazel \
        openjdk-8-jdk \
        && \
    sudo apt-get autoremove -y && \
    echo "Cache Busted: 2018-01-05"

# Add necessary Go build tools
RUN sudo chown $(whoami) $GOPATH -R && \
    go get -u github.com/golang/lint/golint && \
    go get -u golang.org/x/tools/cmd/goimports && \
    go get -u github.com/jstemmer/go-junit-report && \
    sudo curl -L -o $GOPATH/bin/dep https://github.com/golang/dep/releases/download/v0.3.2/dep-linux-amd64 && \
    sudo chmod +x $GOPATH/bin/dep

# Add Node
RUN curl -sL https://deb.nodesource.com/setup_8.x | sudo -E bash - && \
    sudo apt-get install -y nodejs

# Add Yarn
RUN sudo apt-get update && \
    sudo apt-get install apt-transport-https && \
    curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | sudo apt-key add - && \
    echo "deb https://dl.yarnpkg.com/debian/ stable main" | sudo tee /etc/apt/sources.list.d/yarn.list && \
    sudo apt-get update && \
    sudo apt-get install yarn

# Add cypress.io dependencies
RUN sudo apt-get update && \
    sudo apt-get install xvfb libgtk2.0-0 libnotify-dev libgconf-2-4 libnss3 libxss1 libasound2

# Install envsubst
RUN sudo apt-get update && \
    sudo apt-get install -y gettext

# Install gcloud
RUN export CLOUD_SDK_REPO="cloud-sdk-$(lsb_release -c -s)" && \
    echo "deb http://packages.cloud.google.com/apt $CLOUD_SDK_REPO main" | sudo tee -a /etc/apt/sources.list.d/google-cloud-sdk.list && \
    curl https://packages.cloud.google.com/apt/doc/apt-key.gpg | sudo apt-key add - && \
    sudo apt-get update && sudo apt-get install google-cloud-sdk

# Install kubectl
RUN curl -LO https://storage.googleapis.com/kubernetes-release/release/$(curl -s https://storage.googleapis.com/kubernetes-release/release/stable.txt)/bin/linux/amd64/kubectl && \
    chmod +x ./kubectl && \
    sudo mv ./kubectl /usr/local/bin/kubectl && \
    mkdir -p ~/.kube && \
    touch ~/.kube/config