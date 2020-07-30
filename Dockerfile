FROM alpine:3.12
ADD bin/saymyname /bin/
ENTRYPOINT ["/bin/saymyname"]