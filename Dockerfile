FROM alpine:3.14

MAINTAINER Prasad Ghangal<prasad.ghangal@gmail.com>

ADD covaccine-notifier /covaccine-notifier
ENTRYPOINT ["/covaccine-notifier"]
