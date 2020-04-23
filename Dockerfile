# syntax = docker/dockerfile-upstream:1.1.4-experimental

FROM golang:1.13 AS build
ENV GO111MODULE on
ENV GOPROXY https://proxy.golang.org
ENV CGO_ENABLED 0
WORKDIR /tmp
RUN go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.2.5
WORKDIR /src
COPY ./go.mod ./
COPY ./go.sum ./
RUN go mod download
RUN go mod verify
COPY ./ ./
RUN go list -mod=readonly all >/dev/null
RUN ! go mod tidy -v 2>&1 | grep .

FROM build AS manifests-build
RUN controller-gen rbac:roleName=manager-role crd paths="./..." output:rbac:artifacts:config=config/rbac output:crd:artifacts:config=config/crd/bases
FROM scratch AS manifests
COPY --from=manifests-build /src/config/crd /config/crd
COPY --from=manifests-build /src/config/rbac /config/rbac

FROM build AS generate-build
RUN controller-gen object:headerFile=./hack/boilerplate.go.txt paths="./..."
FROM scratch AS generate
COPY --from=generate-build /src/api /api

FROM k8s.gcr.io/hyperkube:v1.17.0 AS release-build
RUN apt update -y \
  && apt install -y curl \
  && curl -LO https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2Fv3.4.0/kustomize_v3.4.0_linux_amd64.tar.gz \
  && tar -xf kustomize_v3.4.0_linux_amd64.tar.gz -C /usr/local/bin \
  && rm kustomize_v3.4.0_linux_amd64.tar.gz
COPY ./config ./config
ARG REGISTRY_AND_USERNAME
ARG NAME
ARG TAG
RUN cd config/manager \
  && kustomize edit set image controller=${REGISTRY_AND_USERNAME}/${NAME}:${TAG} \
  && cd - \
  && kubectl kustomize config/default >/release.yaml
FROM scratch AS release
COPY --from=release-build /release.yaml /release.yaml

FROM build AS binary
RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux go build -ldflags "-s -w" -o /manager
RUN chmod +x /manager

FROM alpine:3.11 AS assets
RUN apk add --no-cache curl
RUN curl -s -o /undionly.kpxe http://boot.ipxe.org/undionly.kpxe
RUN curl -s -o /ipxe.efi http://boot.ipxe.org/ipxe.efi

FROM build AS agent-build
RUN --mount=type=cache,target=/root/.cache/go-build GOOS=linux go build -ldflags "-s -w" -o /agent ./cmd/agent
RUN chmod +x /agent

FROM scratch AS agent
COPY --from=docker.io/autonomy/ca-certificates:v0.1.0 / /
COPY --from=docker.io/autonomy/fhs:v0.1.0 / /
COPY --from=agent-build /agent /agent
ENTRYPOINT [ "/agent" ]

FROM autonomy/tools:v0.1.0 AS initramfs-archive
ENV PATH /toolchain/bin
RUN [ "/toolchain/bin/mkdir", "/bin" ]
RUN [ "ln", "-s", "/toolchain/bin/bash", "/bin/sh" ]
WORKDIR /initramfs
COPY --from=agent /agent ./init
RUN set -o pipefail && find . 2>/dev/null | cpio -H newc -o | xz -v -C crc32 -0 -e -T 0 -z >/initramfs.xz

FROM scratch AS initramfs
COPY --from=initramfs-archive /initramfs.xz /initramfs.xz

FROM scratch AS container
COPY --from=docker.io/autonomy/ca-certificates:v0.1.0 / /
COPY --from=docker.io/autonomy/fhs:v0.1.0 / /
COPY --from=docker.io/autonomy/linux-firmware:v0.2.0 /lib/firmware/bnx2 /lib/firmware/bnx2
COPY --from=docker.io/autonomy/linux-firmware:v0.2.0 /lib/firmware/bnx2x /lib/firmware/bnx2x
COPY --from=assets /undionly.kpxe /var/lib/arges/tftp/undionly.kpxe
COPY --from=assets /undionly.kpxe /var/lib/arges/tftp/undionly.kpxe.0
COPY --from=assets /ipxe.efi /var/lib/arges/tftp/ipxe.efi
COPY --from=initramfs /initramfs.xz /var/lib/arges/env/discovery/initramfs.xz
ADD https://github.com/talos-systems/talos/releases/download/v0.4.1/vmlinuz /var/lib/arges/env/discovery/vmlinuz
COPY --from=binary /manager /manager
ENTRYPOINT [ "/manager" ]
