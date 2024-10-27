import csv

import requests
import time


def get_datastation():
    """
    Fonction principale pour récupérer les données de la station météoFrance de Rennes Gallet (id : 35238003)

    :return: un fichier avec les données demandées
    """

    token = (
        'eyJ4NXQiOiJZV0kxTTJZNE1qWTNOemsyTkRZeU5XTTRPV014TXpjek1UVmhNbU14T1RSa09ETXlOVEE0Tnc9PSIsImtpZCI6ImdhdGV3YXlfY2VydGlmaWNhdGVfYWxpYXMiLCJ0eXAiOiJKV1QiLCJhbGciOiJSUzI1NiJ9.eyJzdWIiOiJteWNlbGl1bUBjYXJib24uc3VwZXIiLCJhcHBsaWNhdGlvbiI6eyJvd25lciI6Im15Y2VsaXVtIiwidGllclF1b3RhVHlwZSI6bnVsbCwidGllciI6IlVubGltaXRlZCIsIm5hbWUiOiJEZWZhdWx0QXBwbGljYXRpb24iLCJpZCI6OTU2MywidXVpZCI6IjQzZGMyOTYwLTMxMzQtNDQ3Mi05NWUwLThjNGI1YTVlZmUyNiJ9LCJpc3MiOiJodHRwczpcL1wvcG9ydGFpbC1hcGkubWV0ZW9mcmFuY2UuZnI6NDQzXC9vYXV0aDJcL3Rva2VuIiwidGllckluZm8iOnsiNTBQZXJNaW4iOnsidGllclF1b3RhVHlwZSI6InJlcXVlc3RDb3VudCIsImdyYXBoUUxNYXhDb21wbGV4aXR5IjowLCJncmFwaFFMTWF4RGVwdGgiOjAsInN0b3BPblF1b3RhUmVhY2giOnRydWUsInNwaWtlQXJyZXN0TGltaXQiOjAsInNwaWtlQXJyZXN0VW5pdCI6InNlYyJ9fSwia2V5dHlwZSI6IlBST0RVQ1RJT04iLCJzdWJzY3JpYmVkQVBJcyI6W3sic3Vic2NyaWJlclRlbmFudERvbWFpbiI6ImNhcmJvbi5zdXBlciIsIm5hbWUiOiJEb25uZWVzUHVibGlxdWVzT2JzZXJ2YXRpb24iLCJjb250ZXh0IjoiXC9wdWJsaWNcL0RQT2JzXC92MSIsInB1Ymxpc2hlciI6ImJhc3RpZW5nIiwidmVyc2lvbiI6InYxIiwic3Vic2NyaXB0aW9uVGllciI6IjUwUGVyTWluIn0seyJzdWJzY3JpYmVyVGVuYW50RG9tYWluIjoiY2FyYm9uLnN1cGVyIiwibmFtZSI6IkRvbm5lZXNQdWJsaXF1ZXNDbGltYXRvbG9naWUiLCJjb250ZXh0IjoiXC9wdWJsaWNcL0RQQ2xpbVwvdjEiLCJwdWJsaXNoZXIiOiJhZG1pbl9tZiIsInZlcnNpb24iOiJ2MSIsInN1YnNjcmlwdGlvblRpZXIiOiI1MFBlck1pbiJ9XSwiZXhwIjoxNzE5NjM5NzkyLCJ0b2tlbl90eXBlIjoiYXBpS2V5IiwiaWF0IjoxNzE0MzgwMTkyLCJqdGkiOiIyMjcwNDMzNS01YjMwLTQ2OTYtODk0YS02NzJjYWVlY2FkOGYifQ==.dd-soN-o7irrLlI16C9JxhAAakHxEqjVdojsp6agfQNrIeARIOrdYvH9xh6JJuBlPIE4WojxZ61jXA9NiWKu1tx8VYPo76Em-3sVKiIXk7sGxkytysnUk-Bkcjf9_M8rjP7BVAnQU7dKDbJ1Q5XJWNuXhBJLVIAbRMnDpfhiHMTxw0Ku0gGj2PDfUrHAiWbUGQLTxMQr0nPGx-YGP99eVHvMMIyqaw2QYlouNIxLk9dLxDQpULwDtSXPbsR4yfK14_puZy_8fyCd_bb6bQgQ9nG1U90cA60JTEIzlCDgiy8X_xbLSPdocNpSgoE3J-_aiok-tReZN_kHPI342oKlFg==')
    ressource = "DPClim/v1/commande-station/horaire?id-station=35238003&date-deb-periode=2023-01-01T00%3A00%3A00Z&date-fin-periode=2023-12-31T00%3A00%3A00Z"
    # requête get pour le departement 35 id_station : 35238003

    reponse = requests.get(f'https://public-api.meteofrance.fr/public/{ressource}',
                           headers={"apikey": f'{token}', "accept": "*/*"})

    print(reponse.headers)

    if reponse.status_code == 202:
        # impression des données json de la réponse dans un dictionnaire
        print("liste envoyée")
        dictionnaire = reponse.json()
        print(dictionnaire)

        id_doc = dictionnaire["elaboreProduitAvecDemandeResponse"]["return"]
        print(id_doc)

        commande = requests.get(f'https://public-api.meteofrance.fr/public/DPClim/v1/commande/fichier?id-cmde={id_doc}')

        while commande.status_code != 201:
            commande = requests.get(
                f'https://public-api.meteofrance.fr/public/DPClim/v1/commande/fichier?id-cmde={id_doc}',
                headers={"apikey": f'{token}', "accept": "*/*"})
            print("je boucle")
            time.sleep(5)

        decoded_content = commande.content.decode('utf-8')

        file = csv.reader(decoded_content.splitlines(), delimiter=';')
        # Specify the filename to save the CSV data
        filename = 'meteofrance.csv'

        # Open the file in write mode and create a CSV writer
        with open(filename, 'w', newline='') as csvfile:
            writer = csv.writer(csvfile)

            # Write each row from the CSV reader to the file
            for row in file:
                writer.writerow(row)

    else:
        print("error")
        return None


if __name__ == "__main__":
    get_datastation()
