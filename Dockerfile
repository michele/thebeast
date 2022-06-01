FROM golang:1.7

RUN curl -s https://glide.sh/get | sh

ADD . /go/src/thebeast

#COPY config.env /go/bin

WORKDIR /go/src/thebeast

RUN glide install

RUN go install thebeast

ENTRYPOINT ["/go/bin/thebeast", "start"]

EXPOSE 8080