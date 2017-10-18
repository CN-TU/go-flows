Feature API
===========


Feature Types
-------------

Valuetypes:

Const:
    Only constant feature
RawPacket:
    Raw input Packet (no feature Values!)
RawFlow:
    Raw input Flow (no feature Values!) (not implemented)
PacketFeature:
    A feature extracted from a single packet. Constants are also allowed. If this is a const and only argument, called on every packet.
FlowFeature:
    A feature extracted from a packet flow. If this is a const and only argument, called on begin of flow (FIXME: might change).
FeatureTypeMatch:
    Either PacketFeature, FlowFeature, or Constant. Return type MUST be the same.
FeatureTypeSelect:
    A selection of RawPacket or RawFlow.
Ellipsis:
    Only allowed as last argument. Represents zero or more repetitions of the next to last argument.


Every feature needs to be assigned an ipfix IE. Least possible specification is just the name, if
it is included in the iana list or in our own specification. If it is contained in a foreign specification,
enterprise number, id, type, and possibly length is also needed. If it is a temporary information element,
it needs a name, a type, and possibly a length.


Value Types
-----------

The follwing list of go types is valid for values (not enforced):

* ``[]byte``
* all ``int`` types
* ``float32`` and ``float64``
* ``bool``
* ``net.HWAddress`` only MAC Adresses allowed
* ``string``
* ``DateTimeSeconds`` (``uint64``)
* ``DateTimeMilliseconds`` (``uint64``)
* ``DateTimeMicroseconds`` (``uint64``)
* ``DateTimeNanoseconds`` (``uint64``)
* ``net.IP`` IPv4 and IPv6 allowed

Upconversion rules
^^^^^^^^^^^^^^^^^^

If multiple different types are encountered, those are upconverted via the provided functions according
to the following rules:

* unsigned types -> ``uint64``
* signed types -> ``int64``
* float -> ``float64`` (everything in ``math`` package is ``float64``)
* datetime -> same datetime
* different datetime -> convert to ``DateTimeNanoseconds``
