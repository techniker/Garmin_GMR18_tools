# Simple tool for talking to the Garmin GMR18 Radome
#
# Thanks: promovicz (https://github.com/promovicz/garmin-radar)
#
# Bjoern Heller <tec(att)sixtopia.net>

import asyncio
import socket
import struct
import json
import paho.mqtt.client as mqtt

class GarminSample:
    def __init__(self, angle, range):
        self.angle = angle
        self.range = range
        self.samples = []

    def set_samples(self, samples):
        self.samples = samples
class GarminRadar:
    def __init__(self, local_address, remote_address, multicast_address, mqtt_broker, mqtt_port=1883):
        self.control_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
        self.multicast_socket = socket.socket(socket.AF_INET, socket.SOCK_DGRAM, socket.IPPROTO_UDP)
        self.multicast_socket.setsockopt(socket.SOL_SOCKET, socket.SO_REUSEADDR, 1)
        self.multicast_socket.bind((local_address, 50100))
        self.multicast_socket.setsockopt(socket.IPPROTO_IP, socket.IP_ADD_MEMBERSHIP,
                                         socket.inet_aton(multicast_address) + socket.inet_aton(local_address))
        self.loop = asyncio.get_event_loop()

        # MQTT setup
        self.mqtt_client = mqtt.Client()
        self.mqtt_broker = mqtt_broker
        self.mqtt_port = mqtt_port
        self.mqtt_topic = "garmin/gmr18radar"
        self.connect_mqtt()

    def connect_mqtt(self):
        self.mqtt_client.connect(self.mqtt_broker, self.mqtt_port, 60)
        self.mqtt_client.loop_start()

    def publish_mqtt(self, topic, message):
        self.mqtt_client.publish(topic, message)

    def power_off(self):
        print("Powering off the radar")
        self.send_control_ushort(0x2b2, 1)

    def power_on(self):
        print("Powering on the radar")
        self.send_control_ushort(0x2b2, 2)

    def set_ftc(self, on):
        print(f"Setting FTC to {on}")
        self.send_control_uchar(0x2b8, 1 if on else 0)

    def set_crosstalk(self, on):
        print(f"Setting CROSSTALK to {on}")
        self.send_control_uchar(0x2b9, 1 if on else 0)

    def set_gain(self, manual, value=0):
        print("Setting GAIN to ", end="")
        if manual:
            print(value)
        else:
            print("AUTO")
            value = 344  # XXX
        self.send_control_uint(0x2b4, value)

    def set_range(self, range_nm):
        range_m = range_nm * 1852.0
        range_val = int(range_m - 1)
        print(f"Setting range to {range_nm} nm ({range_m} m, value {range_val})")
        self.send_control_uint(0x2b3, range_val)

    def send_control_uint(self, frame_type, data):
        frame = struct.pack('>II', frame_type, 4) + struct.pack('>I', data)
        self.control_socket.sendto(frame, (remote_address, 50101))

    def send_control_ushort(self, frame_type, data):
        frame = struct.pack('>II', frame_type, 2) + struct.pack('>H', data)
        self.control_socket.sendto(frame, (remote_address, 50101))

    def send_control_uchar(self, frame_type, data):
        frame = struct.pack('>II', frame_type, 1) + struct.pack('>B', data)
        self.control_socket.sendto(frame, (remote_address, 50101))

    async def handle_frame(self):
        while True:
            data, sender = await self.loop.sock_recvfrom(self.multicast_socket, 1500)
            frame_type = struct.unpack('>I', data[:4])[0]
            if frame_type == 0x2a3:  # frame_type_scanline
                await self.handle_scanline_frame(data)
            elif frame_type == 0x2a5:  # frame_type_status
                self.handle_status_frame(data)
            elif frame_type == 0x2a7:  # frame_type_response
                self.handle_response_frame(data)
            else:
                print(f"Unknown frame type {frame_type}")

    async def handle_scanline_frame(self, data):
        scanline = struct.unpack_from('>IIHHI4B4Bb2B4Bb7B', data, 4)
        angle, range_meters, scan_length_bytes = scanline[2], scanline[4], scanline[3]
        print(f"Scanline: angle {angle} range {range_meters} meters, length {scan_length_bytes}")

        samples = struct.unpack_from(f'>{scan_length_bytes // 4}B', data, 0x36)
        sample_list = [GarminSample((angle * 100) + (i * 25), range_meters) for i in range(4)]
        for i, sample in enumerate(sample_list):
            sample.set_samples(samples[i::4])

        # Publish scanline data to MQTT
        scanline_data = {
            'angle': angle,
            'range_meters': range_meters,
            'samples': samples.tolist()  # Convert samples to list
        }
        self.publish_mqtt(f"{self.mqtt_topic}/scanline", json.dumps(scanline_data))

        def handle_status_frame(self, data):
        status = struct.unpack_from('>HHII', data, 4)
        state, countdown = status[0], status[1]
        if state == 1:
            print(f"Warming up, ready in {countdown}")
        elif state == 3:
            print("Standby")
        elif state == 4:
            print("Active")
        elif state == 5:
            print("Spinup")
        else:
            print(f"Unknown state {state}")
            
        # Publish status data to MQTT
        status_data = {
        'state': state,
        'countdown': countdown
    }
    self.publish_mqtt(f"{self.mqtt_topic}/status", json.dumps(status_data))

    def handle_response_frame(self, data):
        response = struct.unpack_from('>4x4xI4B4Bb7B', data, 4)
        range_meters, gain_mode, gain_level, ftc, crosstalk = response[0], response[1], response[2], response[3], response[4]
        print(f"Range {range_meters} m")
        print(f"Gain {'AUTO' if gain_mode else gain_level}%")
        print(f"FTC {'ON' if ftc else 'OFF'}")
        print(f"Crosstalk {'ON' if crosstalk else 'OFF'}")

        # Publish status data to MQTT
        status_data = {
            'range_meters': range_meters,
            'gain_mode': gain_mode,
            'gain_level': gain_level,
            'ftc': ftc,
            'crosstalk': crosstalk
        }
        self.publish_mqtt(f"{self.mqtt_topic}/status", json.dumps(status_data))

async def read_commands(radar):
    while True:
        cmd = input("Enter a command: ")
        if cmd == 'a':
            radar.power_on()
        elif cmd == 'q':
            radar.power_off()
        elif cmd == 'w':
            radar.set_range(0.25)
        elif cmd == 'e':
            radar.set_range(0.50)
        elif cmd == 'r':
            radar.set_range(1.00)
        elif cmd == 't':
            radar.set_range(3.00)
        elif cmd == 'x':
            radar.set_crosstalk(False)
        elif cmd == 'X':
            radar.set_crosstalk(True)
        elif cmd == 'c':
            radar.set_ftc(False)
        elif cmd == 'C':
            radar.set_ftc(True)
        elif cmd == 's':
            radar.set_gain(False)
        elif cmd == 'd':
            radar.set_gain(True, 0)
        elif cmd == 'f':
            radar.set_gain(True, 25)
        elif cmd == 'g':
            radar.set_gain(True, 50)
        elif cmd == 'h':
            radar.set_gain(True, 75)
        elif cmd == 'j':
            radar.set_gain(True, 100)
        else:
            print("Unknown command.")

if __name__ == "__main__":
    try:
        local_address = "0.0.0.0"
        remote_address = "172.16.2.0"
        multicast_address = "239.254.2.0"
        mqtt_broker = "mqtt.yourbroker.com"  # Replace with your MQTT broker address
        mqtt_port = 1883  # Replace with your MQTT broker port if different

        radar = GarminRadar(local_address, remote_address, multicast_address, mqtt_broker, mqtt_port)

        loop = asyncio.get_event_loop()
        loop.create_task(read_commands(radar))
        loop.create_task(radar.handle_frame())
        loop.run_forever()

    except Exception as e:
        print(f"Exception: {e}")