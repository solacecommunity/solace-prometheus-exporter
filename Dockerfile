FROM amd64/busybox
COPY solace_exporter /bin/solace_exporter
CMD ["/bin/solace_exporter"]