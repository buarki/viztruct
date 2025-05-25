FROM alpine:3.19

WORKDIR /benchmark

# Install required packages
RUN apk add --no-cache git bash bc coreutils curl

# Install Go 1.24.2 manually
RUN curl -L https://go.dev/dl/go1.24.2.linux-amd64.tar.gz -o go.tar.gz && \
    tar -C /usr/local -xzf go.tar.gz && \
    rm go.tar.gz
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=/go
ENV PATH=$PATH:$GOPATH/bin

# Copy the viztruct codebase
COPY . /benchmark/viztruct/

# Build viztruct
WORKDIR /benchmark/viztruct
RUN go build -o /usr/local/bin/viztruct ./cmd/viztruct

# Copy the benchmark script
COPY ./benchmarks/benchmark.sh /benchmark/benchmark.sh
RUN chmod +x /benchmark/benchmark.sh

# Set the working directory
WORKDIR /benchmark

# Set the entry point to the benchmark script
ENTRYPOINT ["/benchmark/benchmark.sh"]
