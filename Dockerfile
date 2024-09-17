FROM scratch

ADD bin/stagger /

ENTRYPOINT [ "/stagger" ]
