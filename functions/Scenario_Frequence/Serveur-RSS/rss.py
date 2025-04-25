from flask import Flask, Response, request, jsonify
from datetime import datetime, timezone
from feedgen.feed import FeedGenerator

from influxdb import InfluxDBClient
from influxdb.exceptions import InfluxDBClientError

# Define InfluxDB connection parameters
INFLUXDB_HOST = 'localhost'
INFLUXDB_PORT = 8086
INFLUXDB_USERNAME = 'your_username'
INFLUXDB_PASSWORD = 'your_password'
INFLUXDB_DATABASE = 'your_database'

# Establish connection to InfluxDB
influx_client = InfluxDBClient(host=INFLUXDB_HOST, port=INFLUXDB_PORT, username=INFLUXDB_USERNAME, password=INFLUXDB_PASSWORD)

# Check if the database exists, create it if not
try:
    databases = influx_client.get_list_database()
    if {'name': INFLUXDB_DATABASE} not in databases:
        influx_client.create_database(INFLUXDB_DATABASE)
except InfluxDBClientError as e:
    print(f"Error: {e}")

# Switch to the specified database
influx_client.switch_database(INFLUXDB_DATABASE)

app = Flask(__name__)

# Create FeedGenerator object
fg = FeedGenerator()
fg.title('Mycelium 3.0')
fg.link(href='https://projets-info.insa-rennes.fr/projets/2022/Mycélium_2.0/', rel='alternate')
fg.description('Flux RSS du projet Mycelium 3.0')
fg.language('fr')
pub_date = datetime.now(timezone.utc).isoformat()  
fg.pubDate(pub_date)
fg.lastBuildDate(pub_date)


@app.route('/rss')
def rss():
    # Retrieve items from InfluxDB
    result = influx_client.query('SELECT * FROM items')
    items = list(result.get_points())
    
    for item in items:
        fe = fg.add_entry()
        fe.title(item['title'])
        fe.link(href=item['link'])
        fe.description(item['description'])
        fe.pubDate(item['pubDate'])  

    rss_content = fg.rss_str(pretty=True)
    return Response(rss_content, mimetype='application/xml')



@app.route('/add_post', methods=['POST'])
def add_post():
    data = request.get_json()
    if data:
        title = data.get('title')
        description = data.get('description')
        if title and description:
            link = data.get('link', '')
            pub_date = datetime.now(timezone.utc).isoformat() 
            new_item = {
                'measurement': 'items',
                'fields': {
                    'title': title,
                    'link': link,
                    'description': description,
                    'pubDate': pub_date
                }
            }
            influx_client.write_points([new_item])
            return jsonify({"message": "Post added successfully!"}), 200
        else:
            return jsonify({"error": "Missing required fields in the request!"}), 400
    else:
        return jsonify({"error": "Invalid JSON data received!"}), 400


if __name__ == '__main__':
    app.run(debug=True,host="0.0.0.0",port=8082)
    # influx_client.drop_database(INFLUXDB_DATABASE) to reset the db after closing if needed 

