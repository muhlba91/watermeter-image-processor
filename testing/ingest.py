import argparse
import paho.mqtt.client as mqtt
import os
import sys

def publish_image(broker_url, topic, image_path):
    if not os.path.isfile(image_path):
        print(f"error: file '{image_path}' not found")
        sys.exit(1)

    with open(image_path, "rb") as f:
        file_content = f.read()
        byte_array = bytearray(file_content)

    client = mqtt.Client(mqtt.CallbackAPIVersion.VERSION2)

    try:
        print(f"connecting to {broker_url}...")
        client.connect(broker_url, 1883, 60)
        client.loop_start()
        
        print(f"publishing {image_path} to topic '{topic}'...")
        result = client.publish(topic, byte_array, qos=1)
        
        result.wait_for_publish(timeout=5)
        print("image sent")
    except Exception as e:
        print(f"failed to send image: {e}")
    finally:
        client.loop_stop()
        client.disconnect()

if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Ingest an image into Mosquitto MQTT broker.")
    parser.add_argument("url", help="Broker URL (e.g., localhost or 10.0.150.32)")
    parser.add_argument("topic", help="MQTT Topic (e.g., tele/water_meter/image)")
    parser.add_argument("path", help="Path to the image file")

    args = parser.parse_args()
    publish_image(args.url, args.topic, args.path)
