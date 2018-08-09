/*
Package packet provides gopacket based input functionality for flows.

The packet package contains everything that is needed to process packets efficiently, like gathering
those from sources, calculating keys, and concurrent flow table processing based on package flows.
Additionally, the base gopacket.Packet interface (Buffer) is extended with additional needed functionality
for packet processing.

Buffer

Buffer holds all necessary interfaces for packet processing. Most features will receive this as
input. All the standard interfaces from gopacket.Packet are provided (See
https://github.com/google/gopacket).

For increased performance, the Link/Network/TransportLayer functions should be used instead of Layer().

Size calculations should use the *Length functions, since those already try to handle all the guess
work that is needed, if something goes wrong (e.g. zero ip.length parameter, truncated packets, ...).

Never hold onto one of these buffers, as they are resued for following packets. To store a packet for
later, the Copy() function must be used. As soon as this copy is not needed any longer, the Recycle()
must be called on this copy. Beware that there is only a limited number of Buffers is available and
this implementation deadlocks if it runs out of Buffers.
*/
package packet
