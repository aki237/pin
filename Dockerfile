FROM alpine:latest

RUN apk add iptables
COPY ./scripts/runhook.sh /
COPY artifacts/linux-amd64/pin-linux-amd64.tar.gz /pin.tar.gz
RUN tar xvf pin.tar.gz && rm pin.tar.gz Readme.md LICENSE

CMD /runhook.sh
