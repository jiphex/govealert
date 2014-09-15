MQTT Mauvealert Protocol
========================

For this project I've had to decide on a protocol for sending Mauve
messages (traditionally Protobuf packets sent over a plain UDP socket) via
an MQTT transport.

The client and server implementations at present rely on the following
standard:

Alerts (*not* AlertUpdates) should be marshalled as either JSON or
protobuf, and published to sub-topics of a "base" topic which can be
configured at runtime (defaults to "govealert")

An alert should be published to the topic (using traditional Mauve 
nomenclature for the Alert fields):

  baseTopic/subject/source/id

I.e, for the following alert and a base topic of "govealert":

  id: heartbeat
  source: bar.example.com
  subject: foo.example.com

The topic would be:

  govealert/foo.example.com/bar.example.com/heartbeat

This does mean duplication of the alert and source fields from inside
the alert to the topic, however such a topic is necessary to allow the
topic to be usefully filtered by other applications which may choose to
consume the feed from MQTT.

For example, an entity may choose to run two independent instances of
traditional MauveAlert, and then may use two instances of the mqtt-receiver
application to publish different alerts to the two different instances based
on topic filters.

  govealert/+/siteA/+ => mauve "A"
  govealert/+/siteB/+ => mauve "B"

Changes to this mechanism should be considered carefully, as these may
require the reimplementation of alert producers/consumers.

Heartbeats
==========

The current *server* implementation contains code to publish an MQTT
"heartbeat" packet periodically the following topic:

  baseTopic/HOSTNAME/$heartbeat

This is a packet containing JSON data which contains (at present):

* The hostname of the MQTT receiver
* The current time as reported by the receiver
* The state of the receiver, which is "true" when running, but will be
  set to "false" by the Will packet which will be automatically published
  to the same topic by the MQTT broker when the connection to it is dropped
  by the receiver.

Heartbeat/Will packets are set to be retained, meaning that it should be
possible for observers to see which MQTT receivers have historically been
active/inactive.
