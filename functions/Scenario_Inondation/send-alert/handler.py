import json

from discord_webhook import DiscordWebhook


def handle(req):
    """
    Envoie une alerte sur discord correspondante au payload
    (attention : l'envoi du message ne fonctionne pas avec Insa wifi)

    :param req: le payload
    :return: rien
    """
    data = json.loads(req)
    alertType = data.get("alertType")
    date = data.get("date")

    if alertType == "floodUnderway":
        print(f"Sending message for undergoing flood on {date}")
        message = 'Date : ' + date + ' Type : ' + alertType + '\n' + 'Flood is underway, do not leave your shelter \n'
    elif alertType == "floodPrediction":
        print(f"Sending message for predicted flood on {date}")
        message = 'Date : ' + date + ' Type : ' + alertType + '\n' + 'Flood is predicted, take shelter \n'
    else:
        print(f"Sending message for unknown event on {date}")
        message = 'Unknown type of alert'

    url = ('https://discord.com/api/webhooks/1183687840907923497'
           '/C52_3uJlQB9w4hylLsVvyY8igw2ZSmogAkXdUUHULK9RHwFwXYGbeavmDFhg-l7hgzMU')
    webhook = DiscordWebhook(url=url, content=message)
    webhook.execute()

