import os
import json
import joblib
import pandas as pd
import asyncio
from datetime import datetime
from influxdb_client import InfluxDBClient, Point
from influxdb_client.client.write_api import SYNCHRONOUS

# Chargement unique au démarrage (Cold Start)
model = joblib.load('./function/rf_fire_model.pkl')
scaler = joblib.load('./function/scaler.pkl')

def handle(req, context):
    payload = req.body
    if isinstance(payload, (bytes, bytearray)):
        payload = payload.decode('utf-8')
    asyncio.run(process(payload))
    return {
        "statusCode": 200,
        "body": "Processed"
    }

async def process(payload):
    try:
        data = json.loads(payload)
        print(f"Received: {data}")

        # Préparation (Ordre des colonnes CRUCIAL)
        features = ['Temperature', 'RH', 'Ws', 'Rain', 'FFMC', 'DMC', 'DC', 'ISI', 'BUI', 'FWI']
        df_input = pd.DataFrame([data], columns=features)
        
        # Prédiction
        X_scaled = scaler.transform(df_input)
        prediction = model.predict(X_scaled)[0]
        probability = model.predict_proba(X_scaled)[0][1]

        print(f"Prediction: {prediction} (Prob: {probability:.2f})")

        # Stockage InfluxDB
        client = InfluxDBClient(
            url=os.environ['INFLUXDB_URL'],
            token=os.environ['INFLUXDB_TOKEN'],
            org=os.environ['INFLUXDB_ORG']
        )
        write_api = client.write_api(write_options=SYNCHRONOUS)
        
        point = Point("fire_prediction") \
            .tag("model", "RandomForest") \
            .field("prediction", int(prediction)) \
            .field("probability", float(probability)) \
            .time(datetime.utcnow())

        write_api.write(bucket=os.environ['INFLUXDB_BUCKET'], record=point)
        print("Saved to InfluxDB")

    except Exception as e:
        print(f"Error: {e}")
