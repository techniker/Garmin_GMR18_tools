# Garmin_GMR18_tools
##Garmin GMR18 Radome Control Tool

This tool offers a simple interface for interacting with the Garmin GMR18 Radome, enabling both local and MQTT-based remote control and configuration. 
It is designed to provide real-time control and status updates for the radar, including power management, range setting, gain adjustment, and more.

Requirements
Python 3.6+
asyncio for asynchronous I/O
socket and struct for network communication
json for data serialization
paho-mqtt Python client library for MQTT communication

## Features

- **Power Control**: Switch the radar power on or off.
- **Range Adjustment**: Modify the radar's scanning range.
- **Gain Management**: Set the radar's gain manually or to auto-adjust.
- **FTC and Crosstalk Management**: Enable or disable Fast Target Tracking (FTC) and Crosstalk.
- **MQTT Integration**: Remotely control the radar and receive status updates via MQTT, enabling integration with automation systems or custom interfaces.
- **Real-Time Data Publishing**: Stream scanline data and radar status updates in real-time over MQTT for external monitoring and analysis.

## Requirements

- Python 3.6 or newer.
- `asyncio` for asynchronous operations.
- `socket` and `struct` for networking.
- `json` for data serialization.
- `paho-mqtt` for MQTT communication.

## Configuration
Network Settings: Configure local_address, remote_address, and multicast_address according to your network setup and radar unit.
- MQTT Settings: Set mqtt_broker and mqtt_port to your MQTT broker's address and port.

MQTT Topics:
- Commands are received on: garmin/radar/command
- Scanline data is published to: garmin/gmr18radar/scanline
- Status updates are published to: garmin/gmr18radar/status

- Issue commands through the command-line interface or by publishing to the garmin/radar/command MQTT topic.
- MQTT Command Structure:
Commands should be in JSON format, for example:

      `{
        "action": "set_range",
        "value": 1.00
      }`

 Supported actions include power_on, power_off, set_range, set_gain, among others, as detailed in the script.

## Contributing
Contributions are welcome. Please submit pull requests for new features, bug fixes and report issues as you come across them. Thanks!


## Acknowledgments

- Special thanks to [promovicz](https://github.com/promovicz/garmin-radar) for their foundational work on Garmin radar communication.
- Developed by Bjoern Heller <tec@sixtopia.net>

## License
Refer to the LICENSE file for more details.
