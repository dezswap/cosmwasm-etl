ARG GO_VERSION="1.21"
ARG BASE_IMAGE="golang:${GO_VERSION}-alpine"

### BUILD
FROM ${BASE_IMAGE} AS build
ARG LIBWASMVM_VERSION=v2.1.4
# required argument: one of("aggregator", "collector", "parser/dex")
ARG APP_PATH

WORKDIR /app

# Create appuser.
RUN adduser -D -g '' appuser
# Install required binaries
RUN apk add --update --no-cache git build-base linux-headers

# Copy app dependencies
COPY go.mod go.mod
COPY go.sum go.sum
COPY Makefile Makefile
# Download all golang package dependencies
RUN make deps

# Copy source files
COPY . .

# See https://github.com/CosmWasm/wasmvm/releases
ADD https://github.com/CosmWasm/wasmvm/releases/download/${LIBWASMVM_VERSION}/libwasmvm_muslc.aarch64.a /lib/libwasmvm_muslc.aarch64.a
ADD https://github.com/CosmWasm/wasmvm/releases/download/${LIBWASMVM_VERSION}/libwasmvm_muslc.x86_64.a /lib/libwasmvm_muslc.x86_64.a
RUN sha256sum /lib/libwasmvm_muslc.aarch64.a | grep 090b97641157fae1ae45e7ed368a1a8c091f3fef67958d3bc7c2fa7e7c54b6b4
RUN sha256sum /lib/libwasmvm_muslc.x86_64.a | grep a4a3d09b36fabb65b119d5ba23442c23694401fcbee4451fe6b7e22e325a4bac
RUN cp /lib/libwasmvm_muslc.`uname -m`.a /lib/libwasmvm_muslc.a

# Build the app
RUN go build -mod=readonly -tags "netgo muslc" \
            -ldflags "-X github.com/cosmos/cosmos-sdk/version.BuildTags='netgo,muslc' \
            -w -s -linkmode=external -extldflags '-Wl,-z,muldefs -static'" \
            -trimpath -o ./main ./cmd/${APP_PATH}

### RELEASE
FROM alpine:latest AS release
WORKDIR /app

# Import the user and group files to run the app as an unpriviledged user
COPY --from=build /etc/passwd /etc/passwd

# Use an unprivileged user
USER appuser
COPY --from=build /app/cmd /app/cmd
# Grab compiled binary from build
COPY --from=build /app/main /app/main

# Set entry point
CMD [ "./main" ]
