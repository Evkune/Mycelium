# noinspection PyUnresolvedReferences
import asyncio
import json
import os
from datetime import datetime, timedelta

import nats
import numpy
import pandas
from keras.api.models import load_model

rainfall_min = 0.0
rainfall_scale = 16.0

waterlevel_min = 0.24
waterlevel_scale = 1.84


def get_sensor_data_sequence(req):
    """
    Lecture des données et stockage dans un DataFrame

    :param req: une liste de 11 payloads avec chacun une date, un niveau d'eau de pluie cumulé
    (rainfall par météoFrance) et le niveau d'eau (waterLevel par VigiCrues)
    :return: return un DataFrame
    """
    data_array = json.loads(req).get("data")

    rainfall_array = []
    water_level_array = []

    # Charger et normaliser les données
    for data in data_array:
        rainfall_array.append((data.get("rainfall") - rainfall_min) / rainfall_scale)
        water_level_array.append((data.get("waterLevel") - waterlevel_min) / waterlevel_scale)

    realtime_historical_data = {'Rainfall (mm)': rainfall_array,
                                'Level (m)': water_level_array}

    realtime_historical_data_df = pandas.DataFrame(realtime_historical_data, columns=['Rainfall (mm)', 'Level (m)'])

    return realtime_historical_data_df


def data_preprocessing(sensor_data_sequence, num_past_hours=10):
    """
    Fonction reprise du papier de recherche
    Prétraitement des données (pour le modèle LSTM)

    :param sensor_data_sequence: un dataframe
    :param num_past_hours: la taille de l'intervalle de temps
    :return: les données prétraitées
    """
    num_features = sensor_data_sequence.shape[1]

    sensor_data_sequence_df = pandas.DataFrame(sensor_data_sequence)
    columns, names = list(), list()

    # Préparation des données historiques (t-n à t-1)
    for n in range(num_past_hours, 0, -1):
        columns.append(sensor_data_sequence_df.shift(n))
        names += [('var%d(t-%d)' % (m + 1, n)) for m in range(num_features)]

    # combine all columns and remove NaN values
    combined_data = pandas.concat(columns, axis=1)
    combined_data.columns = names
    combined_data.dropna(inplace=True)

    # Redimensionnement
    combined_data = numpy.array(combined_data).reshape((combined_data.shape[0], 1, combined_data.shape[1]))

    return combined_data


def data_postprocessing(data):
    """
    Post-traitement des données (déréduction des données)

    :param data: les données prédites
    :return: les données déréduites
    """
    return (data[0][0] * waterlevel_scale) + waterlevel_min


def load_trained_LSTM_model():
    """
    Charger le modèle LSTM entraîné et affiche le résumé du modèle

    :return: le modèle
    """
    model = load_model('./function/model.keras')
    # model = load_model('./model.keras')
    return model


def handle(req):
    """
    Fonction d'arrivée pour les programmes qui tournent avec nats

    :param req: une liste de 11 payloads avec chacun une date, un niveau d'eau de pluie cumulé
    (rainfall par météoFrance) et le niveau d'eau (waterLevel rainFall)
    :return: rien
    """
    asyncio.run(process(req))


def get_prediction_date(req):
    """
    Permet de récupérer la date associée au payload reçu du topic triggerAnalyse

    :param req: une liste de 11 payloads avec chacun une date, un niveau d'eau de pluie (rainfall) et le niveau d'eau (waterlevel)
    :return: la date associée
    """
    data_array = json.loads(req).get("data")

    date_array = []

    for data in data_array:
        date_array.append(datetime.fromisoformat(data.get("date")))

    date_array.sort(reverse=True)
    prediction_date = date_array[0] + timedelta(hours=10)

    return prediction_date


async def process(req):
    """
    Fonction principale qui gère la connexion avec nats et appelle les fonctions pour analyser les données

    :param req: une liste de 11 payloads avec chacun une date, un niveau d'eau de pluie (rainfall) et le niveau d'eau (waterlevel)
    :return: envoie un payload de données prédites sur le topic analysedData
    """
    # Connection au serveur NATS
    nc = await nats.connect(servers=os.environ.get('nats_host'))

    # Chargement du modèle
    model = load_trained_LSTM_model()

    # Lecture des données et stockage dans un DataFrame
    sensor_data_sequence = get_sensor_data_sequence(req)

    # Prétraitement des données (pour le modèle LSTM)
    processed_data = data_preprocessing(sensor_data_sequence)

    print("Receiving data for prediction")

    # Prédiction
    prediction = model.predict(processed_data)

    # Post-traitement des données
    predicted_waterlevel = data_postprocessing(prediction)

    print(f"Predicting water level of {predicted_waterlevel} for {get_prediction_date(req).isoformat()}")

    analyzed_data = {
        'date': get_prediction_date(req).isoformat(),
        'waterLevel': round(predicted_waterlevel, 2)
    }

    json_object = json.dumps(analyzed_data)

    # print(json_object)

    await nc.publish('analysedData', f"{json_object}".encode())
    await nc.flush()
    await nc.close()

