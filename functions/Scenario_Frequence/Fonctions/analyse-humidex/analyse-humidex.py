import json
import math
import time
import os
import paho.mqtt.client as mqtt
import numpy as np
from scipy import stats
import pandas as pd
from datetime import datetime

# Constantes
HISTORY_FILE = "/data/humidex_history.csv"  # Chemin persistant dans un volume monté

# Fonction pour calculer l'indice humidex
def calculate_humidex(temperature, humidity):
    # Calcul de la pression de vapeur d'eau (e)
    e = 6.112 * math.exp((17.67 * temperature) / (temperature + 243.5)) * (humidity / 100.0)
    
    # Calcul de l'indice humidex
    return temperature + (5.0 / 9.0) * (e - 10.0)

# Fonction pour determiner la sensation provoquée en fonction de l'humidex
def get_humidex_sensation(humidex):
    if humidex < 15:
        return "Sensation de frais ou de froid"
    elif humidex >= 15 and humidex < 29:
        return "Sensation de confort"
    elif humidex >= 29 and humidex < 34:
        return "Chaleur : sensation d'inconfort"
    elif humidex >= 34 and humidex < 39:
        return "Chaleur : sensation d'inconfort important"
    elif humidex >= 39 and humidex < 45:
        return "Forte Chaleur : Danger"
    elif humidex >= 45 and humidex < 53:
        return "Très forte chaleur : Danger extrême"
    elif humidex > 54:
        return "Coup de chaleur imminent (danger de mort)"
    return "Erreur HumidexSensation"

# Fonction pour effectuer le test de Student (t-test) pour une valeur
def test_t_student_une_valeur(valeurs_historiques, nouvelle_valeur):
    """
    Effectue un test t de Student pour déterminer si la nouvelle valeur 
    est significativement différente de la moyenne des valeurs historiques.
    """
    # Vérifier s'il y a assez de données
    if len(valeurs_historiques) < 2:
        return 0, 1.0
        
    # Calculer la moyenne et l'écart-type des données historiques
    mean_hist = np.mean(valeurs_historiques)
    std_hist = np.std(valeurs_historiques, ddof=1)  # ddof=1 pour l'échantillon
    
    # Nombre d'observations
    n = len(valeurs_historiques)
    
    # Calculer la statistique t
    if std_hist == 0:  # Éviter la division par zéro
        return 0, 1.0
    
    t_statistic = (nouvelle_valeur - mean_hist) / (std_hist / np.sqrt(n))
    
    # Calculer la p-value pour un test bilatéral
    p_value = 2 * (1 - stats.t.cdf(abs(t_statistic), df=n-1))
    
    return t_statistic, p_value

# Fonction pour envoyer un message MQTT
def send_mqtt_message(message_json, topic, broker_url):
    try:
        client = mqtt.Client("fonction_chaleur")
        host = broker_url.replace("tcp://", "").split(":")[0]
        port = int(broker_url.split(":")[-1])
        
        client.connect(host, port)
        client.publish(topic, json.dumps(message_json))
        print(f"Message publié sur le topic {topic}: {message_json}")
        time.sleep(1)
        client.disconnect()
        return True
    except Exception as e:
        print(f"Erreur lors de l'envoi du message MQTT: {str(e)}")
        return False

# Fonction pour sauvegarder les données dans l'historique
def save_to_history(temperature, humidity, humidex):
    try:
        # S'assurer que le répertoire existe
        os.makedirs(os.path.dirname(HISTORY_FILE), exist_ok=True)
        
        # Créer le fichier s'il n'existe pas
        if not os.path.exists(HISTORY_FILE):
            pd.DataFrame(columns=["timestamp", "temperature", "humidity", "humidex"]).to_csv(HISTORY_FILE, index=False)
        
        # Lire le fichier existant
        history_df = pd.read_csv(HISTORY_FILE)
        
        # Ajouter la nouvelle ligne
        new_row = {
            "timestamp": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
            "temperature": temperature,
            "humidity": humidity,
            "humidex": humidex
        }
        history_df = pd.concat([history_df, pd.DataFrame([new_row])], ignore_index=True)
        
        # Conserver uniquement les 1000 dernières entrées pour éviter une croissance illimitée
        if len(history_df) > 1000:
            history_df = history_df.iloc[-1000:]
            
        history_df.to_csv(HISTORY_FILE, index=False)
        return True
    except Exception as e:
        print(f"Erreur lors de la sauvegarde des données: {str(e)}")
        return False

# Fonction pour charger les données historiques
def load_history():
    try:
        if os.path.exists(HISTORY_FILE):
            return pd.read_csv(HISTORY_FILE)
        return pd.DataFrame(columns=["timestamp", "temperature", "humidity", "humidex"])
    except Exception as e:
        print(f"Erreur lors du chargement des données historiques: {str(e)}")
        return pd.DataFrame(columns=["timestamp", "temperature", "humidity", "humidex"])

# Fonction pour effectuer une analyse statistique des données
def analyze_data(current_temperature, current_humidity, current_humidex):
    history_df = load_history()
    
    if len(history_df) < 10:  # Pas assez de données pour une analyse statistique fiable
        return False, "Pas assez de données historiques pour l'analyse"
    
    # Récupérer les données des dernières 24 heures
    try:
        history_df["timestamp"] = pd.to_datetime(history_df["timestamp"])
        recent_df = history_df[history_df["timestamp"] >= (datetime.now() - pd.Timedelta(days=1))]
    except Exception as e:
        print(f"Erreur lors de la conversion des horodatages: {str(e)}")
        recent_df = history_df  # Utiliser toutes les données en cas d'erreur
    
    if len(recent_df) < 5:  # Pas assez de données récentes
        return False, "Pas assez de données récentes pour l'analyse"
    
    # Récupérer les valeurs récentes pour chaque mesure
    recent_temperatures = recent_df["temperature"].tolist()
    recent_humidities = recent_df["humidity"].tolist()
    recent_humidex = recent_df["humidex"].tolist()
    
    # Effectuer le test t de Student pour chaque mesure
    t_stat_temp, p_value_temp = test_t_student_une_valeur(recent_temperatures, current_temperature)
    t_stat_hum, p_value_hum = test_t_student_une_valeur(recent_humidities, current_humidity)
    t_stat_idx, p_value_idx = test_t_student_une_valeur(recent_humidex, current_humidex)
    
    # On considère qu'il y a une différence significative si l'une des mesures est différente
    significant_difference = (p_value_temp < 0.05) or (p_value_hum < 0.05) or (p_value_idx < 0.05)
    
    # Message d'analyse détaillé
    message = f"Analyse (test t): "
    message += f"Temp (p={p_value_temp:.4f}), "
    message += f"Hum (p={p_value_hum:.4f}), "
    message += f"Idx (p={p_value_idx:.4f}). "
    message += "Différence significative." if significant_difference else "Pas de différence significative."
    
    # Déterminer si on a besoin de plus de données en fonction de la sévérité et des statistiques
    needs_more_data = current_humidex >= 34 or significant_difference
    
    return needs_more_data, message

# Fonction pour changer la période du capteur
def change_sensor_period(needs_more_data):
    period = 120 if needs_more_data else 900  # 2 minutes ou 15 minutes
    
    # Créer le message pour changer la période
    change_period_msg = {
        "periode": period
    }
    
    # Envoyer le message MQTT
    #broker_url = "tcp://192.168.122.61:1883"
    broker_url = os.getenv("MQTT_URL")
    topic = "EM300TH-changePeriode"
    send_mqtt_message(change_period_msg, topic, broker_url)
    
    return period

# Fonction principale traitant les requêtes
def handle(request):
    try:
        # Extraire les données d'entrée
        if isinstance(request, dict):
            data = request
        else:
            # Pour gérer différents formats d'entrée possibles dans un environnement fog
            try:
                data = json.loads(request)
            except:
                return {
                    "status": "error",
                    "message": "Format de données invalide"
                }
        
        # Récupérer les valeurs
        temperature = float(data.get('temperature'))
        humidity = float(data.get('humidity'))
        
        # Calculer l'humidex
        humidex = calculate_humidex(temperature, humidity)
        
        # Déterminer la sensation
        sensation = get_humidex_sensation(humidex)
        
        # Sauvegarder les données dans l'historique
        save_to_history(temperature, humidity, humidex)
        
        # Analyser les données pour déterminer s'il faut augmenter la fréquence
        needs_more_data, analysis_message = analyze_data(temperature, humidity, humidex)
        
        # Changer la période du capteur si nécessaire
        period = change_sensor_period(needs_more_data)
        
        # Préparer la notification
        title = "Humidex Alert"
        description = f"Température: {temperature:.1f}°C, Humidité: {humidity:.1f}%, Humidex: {humidex:.1f}, Sensation: {sensation}. {analysis_message}. Période capteur: {period}s."
        
        notification = {
            "title": title,
            "description": description
        }
        
        # Envoyer la notification via MQTT
        #broker_url = "tcp://10.133.33.52:1883"
        broker_url = os.getenv("MQTT_URL")
        topic = "notification"
        send_mqtt_message(notification, topic, broker_url)
        
        # Réponse de la fonction
        return {
            "status": "success",
            "humidex": humidex,
            "sensation": sensation,
            "analysis": analysis_message,
            "period": period
        }
        
    except Exception as e:
        print(f"Erreur dans la fonction handle: {str(e)}")
        return {
            "status": "error",
            "message": str(e)
        }

# Fonction d'entrée pour les environnements fog/serverless
def main(params):
    return handle(params)