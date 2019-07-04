/*
Package flows provides functionality for implementing table-based merging of
events into flows and extracting features from those events.

The flows package base functionalities for flow-tables, flows, features, and
necessary data types. The most needed functionality are features and data types.

Features

Features are basic structs that receive events and might forward events to dependent features.
This implies that features need at least an argument and return a value. Arguments
and return values have three kinds of types: FeatureTypes, internal data type, and export data type.
FeatureTypes are needed to check which kind of events a feature emits or consumes, which is used
during record/feature instantiation. The internal data type is the concrete type of the data
attached to an event and can be any of the builtin golang types or one of the DateTime-types from
number.go/ipfix package. The export data type is the information element type registered with the
feature. Upon export the internal data type will be converted to the ie-type.

The following FeatureTypes can be used:

 * Const: A constant value (can only be used as argument)
 * RawPacket: An input event from a packet source (can only be used as argument)
 * PacketFeature: A per-packet feature (return or argument)
 * FlowFeature: A per-flow feature (return or argument)
 * MatchType: Return type matches the argument type (Must be used as argument and return)
 * Selection: RawPacket - but filtered
 * Ellipsis: Can only be used as the last argument and means the previous argument type as often a needed

The following structs are available as base for new features:

 * NoopFeature: Features that don't forward events, or hold data. E.g. control or filter features
 * EmptyBaseFeature: Features that don't hold values, but forward events.
 * BaseFeature: Features that hold a value and forward new values to dependent features.
 * MultiBaseFeature: Features with multiple arguments.

Features must be registered with one of the Register* functions.

For examples of features have a look at the already built in features.

Data Types

Data types in this flow implementation are based on the ipfix data types. Convenience functions
are provided to convert between those data types and promote multiple types.

Promotion rules are:

Anything non-64 bit gets converted to 64 bit, and time bases converted to nanoseconds, followed by the following rules

 * If both types are the same ⇒ use this type
 * signed, unsigned ⇒ unsigned
 * signed/unsigned, float ⇒ float
 * number, time ⇒ time

Event Propagation

Events are forwarded in the following way:

 1. Table: find flow according to key, or create new Flow
 2. Flow: forward event to recordlist
 3. Recordlist: Send event to every Record
 4. Record: If the record contains filters do 4a; otherwise continue with 5
 4.a Record, Filters: If the filters haven't been started, start those.
 4.b Record, Filters: Try every filter in order. If everyone acks the event, continue with 5, otherwise continue with 9
 5. Record: If record is not active, call start of every feature.
 6. Record: Call control features and handle stop (call stop event, continue with 9), export (call stop, export, continue with 5) or restart (call stop, continue with 5) conditions
 7. Record: Forward event to every feature
 8. Record: If control feature demands it, handle export (call stop, export, continue with 9), restart (call start, continue with 9)
 9. Flow: If no Record is active, kill the flow

 During Stop control features can prevent an eventual export by calling context.Stop()
*/
package flows
