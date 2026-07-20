// Package lightspeed provides sensor-side integration with OpenShift Lightspeed.
//
// The package consists of two components:
//
//   - updaterImpl: Periodically checks Lightspeed endpoint health and sends status to Central.
//     Accepts LightspeedConfig messages from Central to configure the host URL.
//
//   - querierImpl: Handles LightspeedQueryRequest messages from Central by forwarding
//     them to the Lightspeed API and returning responses.
//
// Both components communicate with Central via the sensor/central gRPC stream using
// the MsgFromSensor/MsgToSensor oneofs defined in internalapi/central/sensor.proto.
package lightspeed
