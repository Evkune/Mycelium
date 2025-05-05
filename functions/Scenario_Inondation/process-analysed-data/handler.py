import asyncio
import json
import os
import aiomqtt as mqtt
from datetime import datetime, timedelta

from influxdb_client import InfluxDBClient, Point
from influxdb_client.client.write_api import SYNCHRONOUS

# Global InfluxDB client instance
db_client = None

def handle(req):
    """
    Triggered by OpenFaaS when a message arrives on MQTT topic 'analysedData'.

    :param req: JSON payload as bytes or str
    :return: None
    """
    # Normalize payload to str
    payload = req.decode('utf-8') if isinstance(req, (bytes, bytearray)) else req
    # Run async pipeline
    asyncio.run(process(payload))

async def process(payload):
    """
    Core logic: parse payload, store to DB, evaluate alerts, publish via MQTT.

    :param payload: JSON payload containing 'date' and 'waterLevel'
    :return: None
    """
    # Parse input JSON
    data = json.loads(payload)
    # Parse ISO date string to datetime
    timestamp = datetime.fromisoformat(data['date'])
    bucket = os.environ['INFLUXDB_BUCKET']

    print(f"Processing analysed data for {timestamp}")

    # 1. Store to InfluxDB in 'predictions' measurement
    export_to_database(bucket, 'predictions', data, timestamp)

    # 2. Evaluate alert condition
    alert_payload = is_there_alert(data)

    # 3. Publish alert via MQTT if needed
    if alert_payload:
        mqtt_url = os.environ.get('MQTT_URL', 'tcp://localhost:1883').replace('tcp://', '')
        broker, port = mqtt_url.split(':')
        async with mqtt.Client(broker, int(port)) as client:
            await client.publish('triggerAlert', json.dumps(alert_payload).encode('utf-8'))

    print("Done processing analysed data.")


def export_to_database(bucket: str, measurement: str, data: dict, timestamp: datetime):
    """
    Write a prediction point to InfluxDB.

    :param bucket: InfluxDB bucket name
    :param measurement: Measurement name (e.g., 'predictions')
    :param data: Dict containing 'waterLevel' and 'date'
    :param timestamp: datetime object
    :return: None
    """
    write_api = connect_db().write_api(write_options=SYNCHRONOUS)
    point = (
        Point(measurement)
        .field('waterLevel', data['waterLevel'])
        .time(timestamp, write_precision='s')
    )
    write_api.write(bucket=bucket, record=point)
    write_api.close()


def is_there_alert(data: dict) -> dict:
    """
    Determine if water level exceeds prediction threshold (1.5m).

    :param data: Dict containing 'waterLevel' and 'date'
    :return: Alert dict or empty dict
    """
    level = data.get('waterLevel')
    print(f"Predicted water level is {level} m")
    if level is not None and level >= 1.5:
        print("Threshold exceeded, triggering floodPrediction alert")
        return {
            'alertType': 'floodPrediction',
            'date': data['date']
        }
    return {}


def connect_db() -> InfluxDBClient:
    """
    Lazy-initialize and return the global InfluxDBClient.
    """
    global db_client
    if db_client is None:
        db_client = InfluxDBClient(
            url=os.environ['INFLUXDB_URL'],
            token=os.environ['INFLUXDB_TOKEN'],
            org=os.environ['INFLUXDB_ORG']
        )
    return db_client

