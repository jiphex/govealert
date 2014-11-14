# Govealert

This is a client for [Bytemark][bm]'s [MauveAlert][mauve] Monitoring service, written in Go.

[MauveAlert][mauve] traditionally uses [Protobuf][protobuf] packets sent over a plain UDP socket. The primary purpose of this client is to take flags specified on the command-line, and to translate these into valid Alert and Update protobuf-encoded packets, and to send these over the network to the right place.

Additionally, this client should provide a Go library that can be used in other Go applications which may want to raise/clear Alerts (see the provided "mauve/" package).

As an additional feature, this client provides support for a new "transport" for Mauve Alerts, transmitting them via an [MQTT][mqtt] broker as an intermediary, with the idea that either the official MauveAlert server would at some point be able to read from the MQTT broker directly, or that another process (already partitially implemented in the _old/ directory) would connect to the MQTT broker and pass off packets to Mauve over UDP directly.

An MQTT transport for alerts would provide the following benefits over the traditional UDP transport:

* Optional SSL encryption
* Confirmed delivery, over TCP using MQTT QOS "ONE" (at-least-once)
* Other applications may interact with the MQTT broker and act on alerts accordingly

This client is *not* intended to be a drop-in replacement for the Ruby `mauvesend` binary included with the `mauvealert` distribution, and the command-line flags will be different.

External dependencies are limited to the following:

* Google's Golang protobuf [library](https://code.google.com/p/goprotobuf/)
* The Eclipse Paho Golang MQTT [library](http://git.eclipse.org/c/paho/org.eclipse.paho.mqtt.golang.git/)
* Packages from the Go standard library

Badges:

[![Build Status](https://travis-ci.org/jiphex/govealert.svg?branch=master)](https://travis-ci.org/jiphex/govealert)

[bm]: http://www.bytemark.co.uk
[mauve]: http://projects.bytemark.co.uk/projects/mauvealert
[protobuf]: https://github.com/google/protobuf
[mqtt]: http://mqtt.org