import grpc
import math
import os
from chirpstack_api import api  # type: ignore
from dotenv import dotenv_values

# Configuration.
config = dotenv_values(".env")
config = {
    **dotenv_values(".env"),  # load shared development variables
    **os.environ,  # override loaded values with environment variables
}

server = config["SERVER"]
api_token = config["API_TOKEN"]
dev_eui = config["DEV_EUI"]
period_sec = config["PERIOD_SEC"]

hex_period = format(int(period_sec), 'x')
len_payload=math.ceil(len(hex_period) / 2)

# Byte conversion and byte order inversion
hex_period = int(period_sec).to_bytes(len_payload, byteorder='big')
hex_period = hex_period.hex()[2:] + hex_period.hex()[:2]
payload = "ff03" + hex_period
payload=bytes.fromhex(payload)


if __name__ == "__main__":
    # Connect without using TLS.
    channel = grpc.insecure_channel(server)

    # Device-queue API client.
    client = api.DeviceServiceStub(channel)

    # Define the API key meta-data.
    auth_token = [("authorization", "Bearer %s" % api_token)]

    # Construct request.
    req = api.EnqueueDeviceQueueItemRequest()
    req.queue_item.confirmed = False
    req.queue_item.data = payload
    req.queue_item.dev_eui = dev_eui
    req.queue_item.f_port = 85

    resp = client.Enqueue(req, metadata=auth_token)

    # Print the downlink id
    print(resp.id)

