FROM scratch

ADD bin/straggler /

ENTRYPOINT [ "/straggler" ]
