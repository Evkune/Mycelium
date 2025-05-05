# noinspection PyUnresolvedReferences
import asyncio
import json
import os
from datetime import datetime, timedelta

import numpy as np
import pandas as pd
import aiomqtt as mqtt
import tensorflow as tf

# Normalization parameters
rainfall_min = 0.0
rainfall_scale = 16.0
waterlevel_min = 0.24
waterlevel_scale = 1.84

def handle(req):
    """
    Entry-point for OpenFaaS via MQTT hook on 'triggerAnalyse'
    
    :param req: 11-payloads list each with a date, waterLevel and rainfall
    :return: None
    """
    print("hello world")
    payload = req.decode('utf-8') if isinstance(req, (bytes, bytearray)) else req
    asyncio.run(process(payload))

async def process(req):
    """
    Core logic: load LSTM model, preprocess data, predict water level, publish results.

    :param req: 11-payloads list each with a date, waterLevel and rainfall
    :return: None
    """
    # 1. Load model & preprocess
    model = load_trained_LSTM_model()
    sensor_df = get_sensor_data_sequence(req)
    processed = data_preprocessing(sensor_df)

    print("Running prediction on processed data")
    prediction = model.predict(processed)
    result = data_postprocessing(prediction)
    pred_date = get_prediction_date(req).isoformat()

    print(f"Predicted water level {result:.2f} at {pred_date}")
    out = {'date': pred_date, 'waterLevel': round(result, 2)}
    msg = json.dumps(out).encode('utf-8')

    # 2. Publish via MQTT
    mqtt_url = os.environ.get('MQTT_URL', 'tcp://10.0.2.15:1883').replace('tcp://', '')
    host, port = mqtt_url.split(':')
    async with mqtt.Client(host, int(port)) as client:
        await client.publish('analysedData', msg)
    print("Published prediction to 'analysedData'.")

def get_sensor_data_sequence(req):
    """
    Build a DataFrame from 11 JSON payloads:
    - create a DataFrame with columns 'Rainfall (mm)' and 'Level (m)'

    :param req: 11-payloads list each with a date, waterLevel and rainfall
    :return: DataFrame with normalized rainfall and waterLevel
    """
    data_array = json.loads(req).get("data", [])
    rainfall_array = []
    water_level_array = []
    for data in data_array:
        rainfall_array.append((data["rainfall"] - rainfall_min) / rainfall_scale)
        water_level_array.append((data["waterLevel"] - waterlevel_min) / waterlevel_scale)
    return pd.DataFrame({
        'Rainfall (mm)': rainfall_array,
        'Level (m)': water_level_array
    })


def data_preprocessing(sensor_df, num_past_hours=10):
    """
    Transform DataFrame into LSTM-ready 3D array
    - Reshape the data to be 3D (samples, time steps, features)
    - Create a new DataFrame with columns 'var1(t-10)', 'var2(t-10)', ..., 'var1(t-1)', 'var2(t-1)'
    - Shift the data to create a time series
    - Remove NaN values
    - Reshape the data to be 3D (samples, time steps, features)
    
    :param sensor_data_sequence: DataFrame with columns 'Rainfall (mm)' and 'Level (m)'
    :param num_past_hours: size of the time interval (default is 10)
    :return: 3D array with shape (samples, time steps, features)
    """
    num_features = sensor_df.shape[1]
    columns, names = [], []
    # Préparation des données historiques (t-n à t-1)
    for n in range(num_past_hours, 0, -1):
        columns.append(sensor_df.shift(n))
        names += [f'var{m+1}(t-{n})' for m in range(num_features)]

    # combine all columns and remove NaN values
    combined = pd.concat(columns, axis=1)
    combined.columns = names
    combined.dropna(inplace=True)

    # Redimensionnement
    arr = combined.to_numpy().reshape((combined.shape[0], 1, combined.shape[1]))

    return arr


def data_postprocessing(predicted):
    """
    Denormalize single LSTM prediction

    :param data: LSTM prediction
    :return: denormalized water level
    """
    return (predicted[0][0] * waterlevel_scale) + waterlevel_min


def load_trained_LSTM_model():
    """
    Load Keras .keras model

    :return: Keras model
    """
    model = tf.keras.models.load_model('./function/model.keras')
    return model


def get_prediction_date(req):
    """
    Determine prediction timestamp: last input date + 10h

    :param req: 11-payloads list each with a date, waterLevel and rainfall
    :return: datetime object representing the prediction date
    """
    dates = [datetime.fromisoformat(d.get("date")) for d in json.loads(req).get("data", [])]
    newest = max(dates)
    return newest + timedelta(hours=10)

