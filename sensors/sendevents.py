import paho.mqtt.client as mqtt
import json
import socket
import logging
import threading
import time
import collections 
import random
from queue import Queue

# Our work queue
workQueue = Queue()

# Holds our array of sensors; we read this
# when the program starts as it won't change
# very often
sensors = {}

# This thread reads the sensor data from the socket, not
# the serial port, using https://github.com/nutechsoftware/ser2sock
# so that the serial data can be used by multiple programs
# concurrently
def readData():
    logging.info("Starting to read data")
    global workQueue

    s = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    s.connect((socket.gethostname(), 10000))
    
    while True:
        full_msg = ''

        while True:
            msg = s.recv(1)
            if len(msg) <= 0:
                break

            full_msg += msg.decode("ascii")
            if not full_msg.endswith("\r"):
                continue
            else:
                break

        if len(full_msg) > 0:
            workQueue.put(full_msg.strip())

# This thread reads the messages that readData() above put
# on the queue and sends them to the MQTT server.
def readQueue():
    def parseLine(line):
        parts = line.split(',')
        if len(parts) != 4:
            return False, parts
        else:
            return True, parts
 
    logging.info("Starting to listen on the queue")
    global workQueue    

    while True:        
        if workQueue.empty() == False:
            line = workQueue.get()        
            logging.info("<--- Pulled %s off the queue", line)
            
            goodLine, parts = parseLine(line)
            if goodLine:
                # Special situation: Zone 008 is "****DISARMED****  Ready to Arm  " which
                # is not a sensor, so for right now we ignore this one
                if parts[1] == "008":
                    continue

                # Now let's find the sensor number in the
                # array...
                sensorName = "Unknown sensor"
                for sensor in sensors:
                    if sensor['zone'] == parts[1]:
                        sensorName = sensor['name']
                # Now we send the timestamp and sensor
                # number to the mqtt server

                ts = round(time.time())
                txline = str(ts) + "," + sensorName + "," + sensor['area']
                logging.info("%s", txline)

                # Set up our mqtt connection
                client = mqtt.Client()
                client.connect('glue', 1883, 60) 
                client.publish('shopmontopic', txline)
                client.disconnect()

if __name__ == "__main__":
    format = "%(asctime)s: %(message)s"
    logging.basicConfig(format=format, level=logging.INFO,
                        datefmt="%H:%M:%S")

    logging.info("Loading dictionary")
    with open('sensors.json') as sensor_file:
        sensors = json.load(sensor_file)

    logging.info("Creating thread to read data")        
    gdThread = threading.Thread(target=readData)
    gdThread.start()

    logging.info("Creating read queue thread")
    rqThread = threading.Thread(target=readQueue)
    rqThread.start()
