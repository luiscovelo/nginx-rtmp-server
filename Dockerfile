ARG NGINX_VERSION=1.23.1
ARG NGINX_RTMP_VERSION=1.2.2

##########################
# Build the nginx image.
FROM amazonlinux:2 as build-nginx
ARG NGINX_VERSION
ARG NGINX_RTMP_VERSION

RUN yum update -y

RUN yum install -y wget tar gzip xz ca-certificates curl make pcre pcre-devel pcre2-devel openssl-devel
RUN yum groupinstall -y 'Development Tools'

WORKDIR /tmp

RUN wget https://nginx.org/download/nginx-${NGINX_VERSION}.tar.gz && \
  tar zxf nginx-${NGINX_VERSION}.tar.gz && \
  rm nginx-${NGINX_VERSION}.tar.gz

RUN wget https://github.com/arut/nginx-rtmp-module/archive/v${NGINX_RTMP_VERSION}.tar.gz && \
  tar zxf v${NGINX_RTMP_VERSION}.tar.gz && \
  rm v${NGINX_RTMP_VERSION}.tar.gz

WORKDIR /tmp/nginx-${NGINX_VERSION}
RUN \
  ./configure \
  --prefix=/usr/local/nginx \
  --add-module=/tmp/nginx-rtmp-module-${NGINX_RTMP_VERSION} \
  --conf-path=/etc/nginx/nginx.conf \
  --with-threads \
  --with-file-aio \
  --with-http_ssl_module \
  --with-debug \
  --with-http_stub_status_module \
  --with-cc-opt="-Wimplicit-fallthrough=0" && \
  make && \
  make install

##########################
# Build the hook server image.
FROM amazonlinux:2 as build-api

RUN yum update -y

RUN yum install -y wget tar gzip xz ca-certificates
RUN yum groupinstall -y 'Development Tools'

RUN wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz && \
    rm -rf /usr/local/go && \
    tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz && \
    rm go1.21.5.linux-amd64.tar.gz

ENV PATH "{$PATH}:/usr/local/go/bin"

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY . ./
RUN CGO_ENABLED=0 go build -o api_done api/main.go

##########################
# Build the release image.
FROM amazonlinux:2

RUN yum update -y

RUN yum install -y wget tar gzip xz ca-certificates curl openssl
RUN yum groupinstall -y 'Development Tools'

# Installing ffmpeg
RUN wget https://www.johnvansickle.com/ffmpeg/old-releases/ffmpeg-6.0.1-amd64-static.tar.xz && \
    rm -rf /usr/local/bin/ffmpeg && \
    tar -C /usr/local/bin -xf ffmpeg-6.0.1-amd64-static.tar.xz && \
    mv /usr/local/bin/ffmpeg-6.0.1-amd64-static/ /usr/local/bin/ffmpeg/ && \
    rm ffmpeg-6.0.1-amd64-static.tar.xz

ENV PATH "{$PATH}:/usr/local/bin/ffmpeg"

COPY --from=build-nginx /usr/local/nginx /usr/local/nginx
COPY --from=build-nginx /etc/nginx /etc/nginx
COPY --from=build-api /app/api_done /usr/local/bin

ENV PATH "${PATH}:/usr/local/nginx/sbin"
ENV LD_LIBRARY_PATH "/usr/lib/"

RUN mkdir -p /opt/data && mkdir /www

COPY ./nginx.conf /etc/nginx/nginx.conf
COPY ./static /www/static

RUN chmod +x /usr/local/bin/api_done
RUN mkdir /tmp/live && chmod 777 /tmp/live

EXPOSE 1935
EXPOSE 8080
EXPOSE 8000

CMD ["sh", "-c", "nginx & api_done & wait"]
