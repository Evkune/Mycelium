import asyncio
import json
import os
import aiomqtt as mqtt
from datetime import datetime, timedelta

from influxdb_client import InfluxDBClient, Point
from influxdb_client.client.write_api import SYNCHRONOUS

# Global InfluxDB client instance
db_client = None
# Date format for incoming data
DATE_FORMAT = '%Y-%m-%dT%H:%M:%S.000Z'

def handle(req):
    """
    Entry point triggered by OpenFaaS when a message arrives on MQTT topic 'rawData'.

    :param req: JSON payload as bytes or str
    :return: None
    """
    
    # Ensure req is a string
    payload = req.decode('utf-8') if isinstance(req, (bytes, bytearray)) else req
    # Delegate to async processing
    asyncio.run(process(payload))

async def process(payload):
    """
    Core logic: parse payload, store to DB, evaluate alerts/analysis, publish results.

    :param payload: JSON payload containing sensor data (rainfall, water level, date)
    :return: None
    """

    # Parse input JSON
    data = json.loads(payload)
    timestamp = datetime.strptime(data['date'], DATE_FORMAT)
    bucket = os.environ['INFLUXDB_BUCKET']

    print(f"Processing data for {timestamp}")
    # 1. Store in InfluxDB
    export_to_database(bucket, 'measures', data, timestamp)

    # 2. Check for alerts
    alert = is_there_alert(data)

    # 3. Query recent data for analysis
    results = query_last_n_hours(bucket, 'measures', timestamp)
    analysis = is_there_analysis(results)

    # 4. Publish messages on MQTT
    mqtt_url = os.environ.get('MQTT_URL', 'tcp://10.0.2.15:1883').replace('tcp://', '')
    broker, port = mqtt_url.split(':')
    async with mqtt.Client(broker, int(port)) as client:
        if alert:
            await client.publish('triggerAlert', json.dumps(alert).encode('utf-8'))
        if analysis:
            await client.publish('triggerAnalyse', json.dumps(analysis).encode('utf-8'))
    # Do not close DB here; rely on process exit :
        # In OpenFaaS, your function container stays alive between invocations (until scaled down or redeployed).
        # This allows us to keep the DB connection alive and reuse it for better performance.
        # Only when the container is terminated will everything (including the DB client) be garbage-collected.
    # close_database_connection()
    print("Done processing data.")

def connect_db():
    """
    Lazily initialize and return an InfluxDBClient.
    This function ensures that the client is created only once and reused for subsequent calls.

    :return: InfluxDBClient instance
    """
    global db_client
    if db_client is None:
        db_client = InfluxDBClient(
            url=os.environ['INFLUXDB_URL'],
            token=os.environ['INFLUXDB_TOKEN'],
            org=os.environ['INFLUXDB_ORG']
        )
    return db_client

def export_to_database(bucket, measurement, data, timestamp):
    """
    Write a data point to InfluxDB.

    :param bucket: InfluxDB bucket name
    :param measurement: Measurement (table) name
    :param data: Dictionary containing 'rainfall' and 'waterLevel'
    :param timestamp: Timestamp for the data point
    :return: None
    """
    write_api = connect_db().write_api(write_options=SYNCHRONOUS)
    point = (
        Point(measurement)
        .field('rainfall', data['rainfall'])
        .field('waterLevel', data['waterLevel'])
        .time(timestamp, write_precision='s')
    )
    write_api.write(bucket=bucket, record=point)
    write_api.close()

def query_last_n_hours(bucket, measurement, end_time, hours=10):
    """
    Query InfluxDB for the last `hours` of data.

    :param bucket: InfluxDB bucket name
    :param measurement: Measurement (table) name
    :param end_time: End time for the query
    :param hours: Number of hours to look back
    :return: Query result (list of records)
    """
    start_time = end_time - timedelta(hours=hours)
    stop_time = end_time + timedelta(seconds=1)  # Mini delta to include the exact end_time record
    query = f'''
        from(bucket: "{bucket}")
        |> range(start: time(v: "{start_time.isoformat()}Z"), stop: time(v: "{stop_time.isoformat()}Z"))
        |> filter(fn: (r) => r._measurement == "{measurement}")
        |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")
    '''  
    query_api = connect_db().query_api()
    return query_api.query(query)

def is_there_alert(data):
    """
    Determine if water level exceeds threshold (1.8m).

    :param data: Dictionary containing 'waterLevel' and 'date'
    :return: JSON with alert type and date if threshold is exceeded, else empty JSON
    """
    print(f"Water level is {data['waterLevel']} m")
    if data['waterLevel'] > 1.8:
        print("Threshold exceeded, triggering alert")
        return {'alertType': 'floodUnderway', 'date': data['date']}
    return {}

def is_there_analysis(results):
    """
    If at least 11 measurements exist, package them for analysis.
    """
    records = []
    for table in results:
        for record in table.records:
            vals = record.values
            records.append({
                'date': record.get_time().isoformat(),
                'rainfall': vals.get('rainfall'),
                'waterLevel': vals.get('waterLevel')
            })
    print(f"Found {len(records)} records in the last period")
    if len(records) > 10:
        print("Enough data for analysis, triggering analysis")
        return {'data': records}
    return {}


def close_database_connection():
    """
    Close the InfluxDB client connection if it exists.

    :return: None
    """
    global client
    if client is not None:
        client.close()
        client = None
        

