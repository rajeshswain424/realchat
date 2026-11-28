# Garden sensor

A Raspberry Pi 4 reads a soil moisture sensor in the vegetable bed.

## Settings
- The sensor is read every 15 minutes.
- Watering starts when moisture drops below 30 percent.
- Readings are published over MQTT to a Mosquitto broker on the topic garden/bed1/moisture.

## Status
The sensor is installed. The watering valve is not connected yet because the 12V adapter is on backorder.
